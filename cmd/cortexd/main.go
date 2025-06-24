package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/loreum-org/cortex/internal/agenthub"
	"github.com/loreum-org/cortex/internal/api"
	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/loreum-org/cortex/internal/economy"
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/pkg/types"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "cortex",
		Short: "Loreum Cortex Node",
		Long:  `Loreum Cortex is a decentralized AI inference network implemented in Go.`,
	}

	var serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the Cortex node",
		Run: func(cmd *cobra.Command, args []string) {
			port, _ := cmd.Flags().GetInt("port")
			p2pPort, _ := cmd.Flags().GetInt("p2p-port")
			bootstrapPeers, _ := cmd.Flags().GetString("bootstrap-peers")
			nodeName, _ := cmd.Flags().GetString("node-name")

			// Generate a node name if not provided
			if nodeName == "" {
				randomBytes := make([]byte, 4)
				rand.Read(randomBytes)
				nodeName = fmt.Sprintf("cortex-node-%x", randomBytes)
			}

			// Parse bootstrap peers
			var bootstrapPeerList []string
			if bootstrapPeers != "" {
				bootstrapPeerList = strings.Split(bootstrapPeers, ",")
			}

			startNode(port, p2pPort, bootstrapPeerList, nodeName)
		},
	}

	// Add flags to serve command
	serveCmd.Flags().Int("port", 8080, "API server port to listen on")
	serveCmd.Flags().Int("p2p-port", 4001, "P2P network port to listen on")
	serveCmd.Flags().String("bootstrap-peers", "", "Comma-separated list of bootstrap peer addresses")
	serveCmd.Flags().String("node-name", "", "Node name for identification (default: auto-generated)")

	var testCmd = &cobra.Command{
		Use:   "test",
		Short: "Test the Cortex node",
		Run: func(cmd *cobra.Command, args []string) {
			runTests()
		},
	}

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(testCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// generateListenAddresses generates listen addresses for the P2P node based on the provided port
func generateListenAddresses(p2pPort int) []string {
	return []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p2pPort),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", p2pPort),
		fmt.Sprintf("/ip6/::/tcp/%d", p2pPort),
		fmt.Sprintf("/ip6/::/udp/%d/quic-v1", p2pPort),
	}
}

func startNode(port, p2pPort int, bootstrapPeers []string, nodeName string) {
	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Generate a key pair for this host
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	// Create network configuration
	netConfig := types.NewNetworkConfig()
	netConfig.PrivateKey = priv
	netConfig.ListenAddresses = generateListenAddresses(p2pPort)
	netConfig.BootstrapPeers = bootstrapPeers

	// Log node information
	log.Printf("Starting Cortex node: %s", nodeName)
	log.Printf("P2P port: %d", p2pPort)
	log.Printf("API port: %d", port)
	if len(bootstrapPeers) > 0 {
		log.Printf("Bootstrap peers: %s", strings.Join(bootstrapPeers, ", "))
	}

	// Create a P2P node
	node, err := p2p.NewP2PNode(netConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Start the P2P node
	if err := node.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Generate an Ed25519 key pair for signing transactions
	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	// Create a consensus service
	cs := consensus.NewConsensusService()

	// Add some validation rules
	cs.ValidationRules = append(cs.ValidationRules, func(tx *types.Transaction) bool {
		return consensus.ValidateTransaction(tx, pubKey)
	})

	// Start processing transactions
	go cs.ProcessTransactions(ctx)

	// Create a solver agent
	solverConfig := &agenthub.SolverConfig{
		PredictorConfig: &agenthub.PredictorConfig{
			Timeout:      10 * time.Second,
			MaxTokens:    1024,
			Temperature:  0.7,
			CacheResults: true,
		},
		// DefaultModel will be auto-selected by SolverAgent
	}
	solverAgent := agenthub.NewSolverAgent(solverConfig)

	// Create a RAG system
	ragSystem := rag.NewRAGSystem(384) // 384 dimensions for the vector space

	// Initialize the Economic Engine
	economicEngine := economy.NewEconomicEngine()
	log.Println("Economic Engine initialized.")

	// Start the Economic Engine maintenance tasks in a goroutine
	go economicEngine.RunMaintenanceTasks(ctx)
	log.Println("Economic Engine maintenance tasks started.")

	// Create some default user and node accounts for demo purposes
	defaultUserID := "user-1"
	defaultUserAddress := "0xUser1Address"
	userAccount, err := economicEngine.CreateUserAccount(defaultUserID, defaultUserAddress)
	if err != nil {
		log.Printf("Warning: Failed to create default user account: %v", err)
	} else {
		log.Printf("Default user account created: %s with address %s", userAccount.ID, userAccount.Address)
	}

	defaultNodeID := "node-1"
	defaultNodeAddress := "0xNode1Address"
	nodeAccount, err := economicEngine.CreateNodeAccount(defaultNodeID, defaultNodeAddress)
	if err != nil {
		log.Printf("Warning: Failed to create default node account: %v", err)
	} else {
		log.Printf("Default node account created: %s with address %s", nodeAccount.ID, nodeAccount.Address)
	}

	// Fund the default user account for testing
	if userAccount != nil {
		// Transfer tokens from network account to user account
		amount := new(big.Int).Mul(big.NewInt(1000), economy.TokenUnit) // 1000 tokens
		tx, err := economicEngine.Transfer("network", defaultUserID, amount, "Initial funding for testing")
		if err != nil {
			log.Printf("Warning: Failed to fund default user account: %v", err)
		} else {
			log.Printf("Transferred %s tokens to user account %s (Transaction ID: %s)", amount.String(), defaultUserID, tx.ID)
		}
	}

	// Create an API server
	server := api.NewServer(node, cs, solverAgent, ragSystem, economicEngine)

	// Start the API server
	go func() {
		address := fmt.Sprintf(":%d", port)
		if err := server.Start(address); err != nil {
			log.Fatal(err)
		}
	}()

	log.Printf("Cortex node started and listening on port %d\n", port)

	// Wait for peers if bootstrap peers are provided
	if len(bootstrapPeers) > 0 {
		// Set a timeout for peer discovery
		peerCtx, peerCancel := context.WithTimeout(ctx, 30*time.Second)
		defer peerCancel()

		log.Println("Waiting for peer connections...")
		err := node.WaitForPeers(peerCtx, 1)
		if err != nil {
			log.Printf("Warning: Peer discovery timed out: %s", err)
		} else {
			log.Println("Successfully connected to peers!")
		}
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	// Stop the API server
	if err := server.Stop(ctx); err != nil {
		log.Printf("Error stopping API server: %s\n", err)
	}

	// Stop the P2P node
	if err := node.Stop(); err != nil {
		log.Printf("Error stopping P2P node: %s\n", err)
	}
}

func runTests() {
	// Create a context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run basic functionality tests
	fmt.Println("Running basic tests...")

	// Test P2P networking
	testP2P(ctx)

	// Test consensus
	testConsensus(ctx)

	// Test agent hub
	testAgentHub(ctx)

	// Test RAG
	testRAG(ctx)

	fmt.Println("All tests passed!")
}

func testP2P(ctx context.Context) {
	// Generate a key pair for this host
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	// Create network configuration
	netConfig := types.NewNetworkConfig()
	netConfig.PrivateKey = priv

	// Create a P2P node
	node, err := p2p.NewP2PNode(netConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Start the P2P node
	if err := node.Start(ctx); err != nil {
		log.Fatal(err)
	}

	// Test publish/subscribe
	_, err = node.JoinTopic("test-topic")
	if err != nil {
		log.Fatal(err)
	}

	// Wait for a bit
	time.Sleep(1 * time.Second)

	// Stop the P2P node
	if err := node.Stop(); err != nil {
		log.Fatal(err)
	}
}

func testConsensus(ctx context.Context) {
	// Generate an Ed25519 key pair for signing transactions
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	// Create a consensus service
	cs := consensus.NewConsensusService()

	// Add some validation rules
	cs.ValidationRules = append(cs.ValidationRules, func(tx *types.Transaction) bool {
		return consensus.ValidateTransaction(tx, pubKey)
	})

	// Create a transaction
	tx, err := consensus.CreateTransaction("test-data", []string{}, privKey)
	if err != nil {
		log.Fatal(err)
	}

	// Add the transaction to the pool
	if err := cs.AddTransaction(tx); err != nil {
		log.Fatal(err)
	}
}

func testAgentHub(ctx context.Context) {
	// Create a solver agent
	solverConfig := &agenthub.SolverConfig{
		PredictorConfig: &agenthub.PredictorConfig{
			Timeout:      10 * time.Second,
			MaxTokens:    1024,
			Temperature:  0.7,
			CacheResults: true,
		},
		DefaultModel: "default",
	}
	solverAgent := agenthub.NewSolverAgent(solverConfig)

	// Create a query
	query := &types.Query{
		ID:        "test-query",
		Text:      "What is the meaning of life?",
		Type:      "question",
		Metadata:  make(map[string]string),
		Timestamp: time.Now().Unix(),
	}

	// Process the query
	response, err := solverAgent.Process(ctx, query)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Response: %s\n", response.Text)
}

func testRAG(ctx context.Context) {
	// Create a RAG system
	ragSystem := rag.NewRAGSystem(384) // 384 dimensions for the vector space

	// Add a document
	err := ragSystem.AddDocument(ctx, "The meaning of life is 42.", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Query the RAG system
	response, err := ragSystem.Query(ctx, "What is the meaning of life?")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("RAG Response: %s\n", response)
}

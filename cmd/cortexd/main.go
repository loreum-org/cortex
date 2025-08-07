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
	"github.com/loreum-org/cortex/internal/agents"
	"github.com/loreum-org/cortex/internal/api"
	"github.com/loreum-org/cortex/internal/consensus"
	"github.com/loreum-org/cortex/internal/economy"
	"github.com/loreum-org/cortex/internal/p2p"
	"github.com/loreum-org/cortex/internal/rag"
	"github.com/loreum-org/cortex/internal/services"
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
	serveCmd.Flags().Int("port", 4891, "API server port to listen on")
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
		// Skip signature validation for economic transactions for now
		if tx.Type == types.TransactionTypeEconomic {
			return true
		}
		return consensus.ValidateTransaction(tx, pubKey)
	})

	// Initialize reputation for this node using its libp2p peer ID
	nodeID := node.Host.ID().String()
	cs.ReputationManager.SetScore(nodeID, 7.5) // This node starts with good reputation

	// Start processing transactions
	go cs.ProcessTransactions(ctx)

	// Solver agent configuration will be handled by the AgentRegistry

	// Create a RAG system with the actual node ID for context tracking
	ragSystem := rag.NewRAGSystemWithNodeID(384, nodeID) // 384 dimensions for the vector space

	// Initialize the Economic Engine
	economicEngine := economy.NewEconomicEngine()
	log.Println("Economic Engine initialized.")

	// Initialize Economic Storage (using in-memory fallback for demo)
	// In production, this would use actual SQL and Redis configurations
	sqlConfig := types.NewStorageConfig("mysql")
	redisConfig := types.NewStorageConfig("redis")

	// For demo purposes, we'll continue without storage if databases aren't available
	// economicStorage, err := storage.NewEconomicStorage(sqlConfig, redisConfig)
	// if err != nil {
	//     log.Printf("Warning: Failed to initialize economic storage: %v", err)
	//     log.Println("Continuing with in-memory storage only")
	// } else {
	//     economicEngine.SetStorage(economicStorage)
	//     log.Println("Economic storage initialized and connected.")
	// }
	log.Printf("Economic storage config prepared (SQL: %s:%d, Redis: %s:%d)",
		sqlConfig.Host, sqlConfig.Port, redisConfig.Host, redisConfig.Port)

	// Create economic consensus bridge
	economicBridge := consensus.NewEconomicBridge(cs, economicEngine)
	economicEngine.SetConsensusBridge(economicBridge)
	log.Println("Economic consensus bridge created and connected.")

	// Create economic P2P broadcaster
	economicBroadcaster := p2p.NewEconomicBroadcaster(node, nodeName)
	economicBridge.SetEconomicBroadcaster(economicBroadcaster)
	log.Println("Economic P2P broadcaster created and connected.")

	// Start economic consensus monitoring
	go economicBridge.MonitorFinalization(ctx)
	log.Println("Economic consensus monitoring started.")

	// Start the Economic Engine maintenance tasks in a goroutine
	go economicEngine.RunMaintenanceTasks(ctx)
	log.Println("Economic Engine maintenance tasks started.")

	// Initialize and start the automatic reward distributor
	economicEngine.InitRewardDistributor(economy.RewardDistributorConfig{})
	economicEngine.StartRewardDistributor()
	log.Println("Automatic reward distributor started.")

	// Create some default user and node accounts for demo purposes
	defaultUserID := "user-1"
	defaultUserAddress := "0xUser1Address"
	userAccount, err := economicEngine.CreateUserAccount(defaultUserID, defaultUserAddress)
	if err != nil {
		log.Printf("Warning: Failed to create default user account: %v", err)
	} else {
		log.Printf("Default user account created: %s with address %s", userAccount.ID, userAccount.Address)
	}

	// Use the actual libp2p peer ID for the node account
	nodePeerID := node.Host.ID().String()
	nodeAccount, err := economicEngine.CreateNodeAccount(nodePeerID)
	if err != nil {
		log.Printf("Warning: Failed to create node account: %v", err)
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

	// Agent connections will be handled by the AgentRegistry

	// Initialize AGI blockchain recorder
	agiRecorder := consensus.NewAGIRecorder(cs, nodeID)
	cs.AGIRecorder = agiRecorder

	// Start AGI recorder
	if err := agiRecorder.Start(ctx); err != nil {
		log.Printf("Warning: Failed to start AGI recorder: %v", err)
	} else {
		log.Println("AGI blockchain recorder started.")
	}

	// Connect AGI system with blockchain recorder
	if ragSystem.ContextManager != nil {
		agiSystem := ragSystem.ContextManager.GetAGISystem()
		if agiSystem != nil {
			agiSystem.SetAGIRecorder(agiRecorder)
			log.Println("Connected AGI system with blockchain recorder.")
		}
	}

	// Economic engine and RAG system connections will be handled by the AgentRegistry

	// Create service registry manager
	serviceRegistry := services.NewServiceRegistryManager(nodePeerID, cs, node)
	log.Printf("Service registry manager initialized for node %s", nodePeerID[:12]+"...")

	// Create an event-driven API server (the only server we need)
	server := api.NewEventDrivenServer(node, cs, ragSystem, economicEngine, serviceRegistry, nil)

	// Start the event-driven API server
	go func() {
		address := fmt.Sprintf(":%d", port)
		log.Printf("Starting event-driven API server on %s", address)
		if err := server.Start(address); err != nil {
			log.Fatal(err)
		}
	}()

	log.Printf("🚀 Cortex node started successfully!")
	log.Printf("📡 API server listening on port %d", port)
	log.Printf("🔗 P2P node listening on port %d", p2pPort)
	log.Printf("🧠 Event-driven architecture initialized")
	log.Printf("💬 WebSocket endpoint: ws://localhost:%d/ws", port)

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

	select {
	case sig := <-sigCh:
		log.Printf("Received signal: %v", sig)
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	log.Println("Shutting down...")

	// Cancel context to stop all goroutines
	cancel()

	// Create a timeout context for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop the API server with timeout
	done := make(chan error, 1)
	go func() {
		done <- server.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("Error stopping API server: %s\n", err)
		}
	case <-shutdownCtx.Done():
		log.Println("API server shutdown timed out")
	}

	// Stop reward distributor
	economicEngine.StopRewardDistributor()

	// Stop the P2P node with timeout
	go func() {
		done <- node.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("Error stopping P2P node: %s\n", err)
		}
	case <-shutdownCtx.Done():
		log.Println("P2P node shutdown timed out")
	}

	log.Println("Shutdown complete")
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
	testAgents(ctx)

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

func testAgents(ctx context.Context) {
	// Create a solver agent
	solverConfig := &agents.SolverConfig{
		PredictorConfig: &agents.PredictorConfig{
			Timeout:      10 * time.Second,
			MaxTokens:    1024,
			Temperature:  0.7,
			CacheResults: true,
		},
		DefaultModel: "default",
	}
	solverAgent := agents.NewSolverAgent(solverConfig)

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

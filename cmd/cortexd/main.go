package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/loreum-org/cortex/internal/agenthub"
	"github.com/loreum-org/cortex/internal/api"
	"github.com/loreum-org/cortex/internal/consensus"
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
			startNode()
		},
	}

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

func startNode() {
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
		DefaultModel: "default",
	}
	solverAgent := agenthub.NewSolverAgent(solverConfig)

	// Create a RAG system
	ragSystem := rag.NewRAGSystem(384) // 384 dimensions for the vector space

	// Create an API server
	server := api.NewServer(node, cs, solverAgent, ragSystem)

	// Start the API server
	go func() {
		if err := server.Start(":8080"); err != nil {
			log.Fatal(err)
		}
	}()

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

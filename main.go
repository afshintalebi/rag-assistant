package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/afshintalebi/rag-assistant/internal/agent"
	"github.com/afshintalebi/rag-assistant/internal/api"
	"github.com/afshintalebi/rag-assistant/internal/config"
	"github.com/afshintalebi/rag-assistant/internal/ingest"
	"github.com/afshintalebi/rag-assistant/internal/llm"
	"github.com/afshintalebi/rag-assistant/internal/memory"
	"github.com/afshintalebi/rag-assistant/internal/router"
	"github.com/afshintalebi/rag-assistant/internal/vectordb"
)

func main() {
	cfg := config.LoadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Initialize Vector DB
	vectorDB, err := vectordb.NewQdrantDB(ctx, cfg.QdrantHost, cfg.QdrantPort, "documents")
	if err != nil {
		log.Fatalf("Failed to initialize Vector Database: %v", err)
	}
	log.Println("Successfully connected to Qdrant")

	// 2. Initialize Redis Memory DB
	memoryDB, err := memory.NewRedisMemory(cfg.RedisAddr)
	if err != nil {
		log.Fatalf("Failed to initialize Redis Memory Database: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// 3. Initialize Core Clients & Processors
	llmClient := llm.NewOpenAIClient(cfg.OpenAIAPIKey)
	processor := ingest.NewProcessor(llmClient, vectorDB)

	// 4. Initialize the AI Agent
	aiAgent := agent.NewAIAgent(llmClient, vectorDB, memoryDB, cfg.OpenAIAPIKey)

	// 5. Setup Handler and Router
	handler := api.NewHandler(llmClient, vectorDB, processor, aiAgent)
	r := router.SetupRouter(handler)

	// 6. Start Server
	serverAddr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server is starting on port %s...", cfg.Port)
	if err := r.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
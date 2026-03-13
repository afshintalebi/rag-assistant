package ingest

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/afshintalebi/rag-assistant/internal/llm"
	"github.com/afshintalebi/rag-assistant/internal/vectordb"
)

type Processor struct {
	llmClient *llm.OpenAIClient
	vectorDB  *vectordb.QdrantDB
}

// NewProcessor creates a new ingestion processor.
func NewProcessor(llmClient *llm.OpenAIClient, vectorDB *vectordb.QdrantDB) *Processor {
	return &Processor{
		llmClient: llmClient,
		vectorDB:  vectorDB,
	}
}

// ProcessConcurrently takes a large slice of chunks, batches them, and processes them concurrently.
func (p *Processor) ProcessConcurrently(ctx context.Context, chunks []string) error {
	batchSize := 10 // Process 10 chunks per API call to avoid token limits
	var wg sync.WaitGroup
	errCh := make(chan error, len(chunks)/batchSize+1)

	for i := 0; i < len(chunks); i += batchSize {
		end := i + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		wg.Add(1)

		// Spawn a Goroutine for each batch
		go func(b []string) {
			defer wg.Done()

			// 1. Generate Embeddings for the batch
			vectors, err := p.llmClient.GenerateEmbeddings(ctx, b)
			if err != nil {
				errCh <- fmt.Errorf("embedding error: %w", err)
				return
			}

			// 2. Upsert to Qdrant
			err = p.vectorDB.Upsert(ctx, b, vectors)
			if err != nil {
				errCh <- fmt.Errorf("qdrant upsert error: %w", err)
				return
			}
			
			log.Printf("Successfully processed a batch of %d chunks", len(b))
		}(batch)
	}

	// Wait for all Goroutines to finish
	wg.Wait()
	close(errCh)

	// Check if any errors occurred during concurrent processing
	for err := range errCh {
		if err != nil {
			return fmt.Errorf("concurrent processing failed: %w", err)
		}
	}

	return nil
}
package vectordb

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

type QdrantDB struct {
	client         *qdrant.Client
	collectionName string
}

// NewQdrantDB establishes a gRPC connection to Qdrant and ensures the collection exists.
func NewQdrantDB(ctx context.Context, host string, portStr string, collectionName string) (*QdrantDB, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant port: %w", err)
	}

	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to qdrant: %w", err)
	}

	db := &QdrantDB{
		client:         client,
		collectionName: collectionName,
	}

	// OpenAI's text-embedding-3-small outputs 1536 dimensions
	err = db.initCollection(ctx, 1536)
	if err != nil {
		return nil, err
	}

	return db, nil
}

// initCollection checks if the collection exists, creates it if it doesn't.
func (db *QdrantDB) initCollection(ctx context.Context, vectorSize uint64) error {
	exists, err := db.client.CollectionExists(ctx, db.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if !exists {
		err = db.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: db.collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     vectorSize,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}
	return nil
}

// Upsert stores text chunks and their vectors in Qdrant.
func (db *QdrantDB) Upsert(ctx context.Context, texts []string, vectors [][]float32) error {
	var points []*qdrant.PointStruct

	for i, vec := range vectors {
		pointId := uuid.New().String()
		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewIDUUID(pointId),
			Vectors: qdrant.NewVectorsDense(vec),
			Payload: qdrant.NewValueMap(map[string]any{
				"text": texts[i],
			}),
		})
	}

	_, err := db.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: db.collectionName,
		Points:         points,
	})

	return err
}

// Search performs a similarity search and returns the top matching text chunks.
func (db *QdrantDB) Search(ctx context.Context, queryVector []float32, limit uint64) ([]string, error) {
	searchResults, err := db.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: db.collectionName,
		Query:          qdrant.NewQueryDense(queryVector),
		WithPayload:    qdrant.NewWithPayload(true),
		Limit:          &limit,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}

	var results []string
	for _, point := range searchResults {
		if textVal, ok := point.Payload["text"]; ok {
			results = append(results, textVal.GetStringValue())
		}
	}

	return results, nil
}
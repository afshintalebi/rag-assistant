package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
}

// NewOpenAIClient initializes a new OpenAI API client.
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		client: openai.NewClient(apiKey),
	}
}

// GenerateEmbeddings converts a list of strings into vectors using text-embedding-3-small.
func (o *OpenAIClient) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.SmallEmbedding3,
	}

	resp, err := o.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	vectors := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		vectors[i] = data.Embedding
	}

	return vectors, nil
}

// GenerateRAGResponse generates a response strictly based on the provided context.
func (o *OpenAIClient) GenerateRAGResponse(ctx context.Context, prompt string, contextText string) (string, error) {
	systemPrompt := fmt.Sprintf(
		"You are a helpful AI research assistant. Answer the user's question based ONLY on the provided context.\n\nContext:\n%s",
		contextText,
	)

	req := openai.ChatCompletionRequest{
		Model:       openai.GPT4oMini,
		Temperature: 0.2, // Low temperature for more factual answers
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	resp, err := o.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
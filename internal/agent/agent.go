package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/afshintalebi/rag-assistant/internal/llm"
	"github.com/afshintalebi/rag-assistant/internal/memory"
	"github.com/afshintalebi/rag-assistant/internal/vectordb"
)

type AIAgent struct {
	llmClient *llm.OpenAIClient
	vectorDB  *vectordb.QdrantDB
	memoryDB  *memory.RedisMemory
	rawClient *openai.Client
}

func NewAIAgent(llmClient *llm.OpenAIClient, vectorDB *vectordb.QdrantDB, memoryDB *memory.RedisMemory, apiKey string) *AIAgent {
	return &AIAgent{
		llmClient: llmClient,
		vectorDB:  vectorDB,
		memoryDB:  memoryDB,
		rawClient: openai.NewClient(apiKey),
	}
}

type SearchArguments struct {
	Query string `json:"query"`
}

// ChatStream handles the Agentic logic and returns a channel that streams the final response token by token.
func (a *AIAgent) ChatStream(ctx context.Context, sessionID string, userPrompt string) (<-chan string, error) {
	history, _ := a.memoryDB.GetHistory(ctx, sessionID)

	systemMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "You are an intelligent AI assistant. Use the 'search_local_documents' tool ONLY when the user asks about specific information that might be in their uploaded files or scraped websites. Keep answers concise.",
	}
	userMsg := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userPrompt,
	}

	messages := []openai.ChatCompletionMessage{systemMsg}
	messages = append(messages, history...)
	messages = append(messages, userMsg)

	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "search_local_documents",
				Description: "Search the local vector database for specific context based on user uploads.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "The specific keyword or question to search in the database."
						}
					},
					"required": ["query"]
				}`),
			},
		},
	}

	req := openai.ChatCompletionRequest{
		Model:    openai.GPT4oMini,
		Messages: messages,
		Tools:    tools,
	}

	// 1. First Call: Check if a tool needs to be called (Non-streaming)
	resp, err := a.rawClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("initial llm call failed: %w", err)
	}

	assistantMsg := resp.Choices[0].Message
	a.memoryDB.AddMessage(ctx, sessionID, userMsg)

	// 2. Execute tool if requested
	if len(assistantMsg.ToolCalls) > 0 {
		toolCall := assistantMsg.ToolCalls[0]
		var args SearchArguments
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
			
			// Retrieve context from Vector DB
			vectors, _ := a.llmClient.GenerateEmbeddings(ctx, []string{args.Query})
			searchResults, _ := a.vectorDB.Search(ctx, vectors[0], 3)
			contextText := strings.Join(searchResults, "\n\n---\n\n")

			messages = append(messages, assistantMsg)
			toolResultMsg := openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    fmt.Sprintf("Database Search Results:\n%s", contextText),
				ToolCallID: toolCall.ID,
			}
			messages = append(messages, toolResultMsg)
		}
	} else {
		// No tool needed, append the assistant's thinking path (if any)
		messages = append(messages, assistantMsg)
	}

	// 3. Second Call: Generate the final answer (Streaming)
	req.Messages = messages
	req.Tools = nil // Remove tools to force a final text answer
	req.Stream = true

	stream, err := a.rawClient.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat stream: %w", err)
	}

	// Create a channel to send tokens back to the HTTP handler
	tokenChan := make(chan string)
	var fullResponse strings.Builder

	// Goroutine to read from OpenAI stream and write to our channel
	go func() {
		defer stream.Close()
		defer close(tokenChan)

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				// Stream finished, save the complete response to memory
				finalMsg := openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: fullResponse.String(),
				}
				a.memoryDB.AddMessage(context.Background(), sessionID, finalMsg)
				return
			}
			if err != nil {
				return
			}

			// Extract the token and send it to the channel
			token := response.Choices[0].Delta.Content
			if token != "" {
				fullResponse.WriteString(token)
				tokenChan <- token
			}
		}
	}()

	return tokenChan, nil
}
package api

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly/v2"
	"github.com/ledongthuc/pdf"

	"github.com/afshintalebi/rag-assistant/internal/agent"
	"github.com/afshintalebi/rag-assistant/internal/ingest"
	"github.com/afshintalebi/rag-assistant/internal/llm"
	"github.com/afshintalebi/rag-assistant/internal/vectordb"
)

type Handler struct {
	llmClient *llm.OpenAIClient
	vectorDB  *vectordb.QdrantDB
	processor *ingest.Processor
	aiAgent   *agent.AIAgent
}

func NewHandler(llmClient *llm.OpenAIClient, vectorDB *vectordb.QdrantDB, processor *ingest.Processor, aiAgent *agent.AIAgent) *Handler {
	return &Handler{
		llmClient: llmClient,
		vectorDB:  vectorDB,
		processor: processor,
		aiAgent:   aiAgent,
	}
}

// --- Request Structs ---

type ChatRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Prompt    string `json:"prompt" binding:"required"`
}

type ScrapeRequest struct {
	URL string `json:"url" binding:"required,url"`
}

// --- Endpoints ---

// HandleChat retrieves relevant context from Qdrant and generates a RAG-based response.
// func (h *Handler) HandleChat(c *gin.Context) {
// 	var req ChatRequest
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format. 'session_id' and 'prompt' are required."})
// 		return
// 	}

// 	ctx := c.Request.Context()

// 	vectors, err := h.llmClient.GenerateEmbeddings(ctx, []string{req.Prompt})
// 	if err != nil || len(vectors) == 0 {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to embed prompt"})
// 		return
// 	}

// 	limit := uint64(5) // Retrieve top 5 matches
// 	matchedTexts, err := h.vectorDB.Search(ctx, vectors[0], limit)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search vector database"})
// 		return
// 	}

// 	response, err := h.aiAgent.Chat(ctx, req.SessionID, req.Prompt)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI Agent encountered an error: " + err.Error()})
// 		return
// 	}

// 	/* 
// 	contextText := strings.Join(matchedTexts, "\n\n---\n\n")
	
// 	response, err := h.llmClient.GenerateRAGResponse(ctx, req.Prompt, contextText)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate AI response"})
// 		return
// 	} */

// 	c.JSON(http.StatusOK, gin.H{
// 		"response": response,
// 		"sources":  matchedTexts,
// 	})
// }

func (h *Handler) HandleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format. 'session_id' and 'prompt' are required."})
		return
	}

	// Set headers for SSE (Server-Sent Events)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ctx := c.Request.Context()

	// Get the streaming channel from the Agent
	tokenChan, err := h.aiAgent.ChatStream(ctx, req.SessionID, req.Prompt)
	if err != nil {
		c.SSEvent("error", "AI Agent encountered an error: "+err.Error())
		return
	}

	// Stream tokens as they arrive
	c.Stream(func(w io.Writer) bool {
		token, ok := <-tokenChan
		if !ok {
			// Channel closed (stream finished)
			c.SSEvent("done", "[DONE]")
			return false
		}

		// Send token to the client
		c.SSEvent("message", token)
		return true
	})
}


// HandleScrape visits a URL, extracts text from <p> tags, and ingests it concurrently.
func (h *Handler) HandleScrape(c *gin.Context) {
	var req ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL provided"})
		return
	}

	collector := colly.NewCollector()
	var scrapedText strings.Builder

	// Extract text from paragraph tags
	collector.OnHTML("p", func(e *colly.HTMLElement) {
		text := strings.TrimSpace(e.Text)
		if len(text) > 20 {
			scrapedText.WriteString(text)
			scrapedText.WriteString("\n")
		}
	})

	err := collector.Visit(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scrape the website"})
		return
	}

	chunks := ingest.ChunkText(scrapedText.String(), 500) // Minimum 500 characters per chunk

	if len(chunks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No meaningful text found on the page"})
		return
	}

	// Process concurrently
	err = h.processor.ProcessConcurrently(c.Request.Context(), chunks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process and save website data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Website successfully scraped and ingested",
		"url":     req.URL,
		"chunks":  len(chunks),
	})
}

// HandleUploadPDF reads an uploaded PDF file, extracts text, and ingests it concurrently.
func (h *Handler) HandleUploadPDF(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload failed or missing 'file' field"})
		return
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer src.Close()

	// Read file into memory (required for the PDF reader)
	content, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file content"})
		return
	}

	// Initialize PDF reader
	pdfReader, err := pdf.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse PDF file"})
		return
	}

	var extractedText strings.Builder
	numPages := pdfReader.NumPage()

	// Extract text page by page
	for i := 1; i <= numPages; i++ {
		page := pdfReader.Page(i)
		if page.V.IsNull() {
			continue
		}
		text, _ := page.GetPlainText(nil)
		extractedText.WriteString(text)
		extractedText.WriteString("\n")
	}

	chunks := ingest.ChunkText(extractedText.String(), 500)

	if len(chunks) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No text could be extracted from the PDF"})
		return
	}

	// Process concurrently
	err = h.processor.ProcessConcurrently(c.Request.Context(), chunks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process and save PDF data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "PDF successfully uploaded and ingested",
		"filename": file.Filename,
		"chunks":   len(chunks),
	})
}

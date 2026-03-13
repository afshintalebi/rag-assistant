# 🚀 Go Agentic RAG Assistant

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)
![OpenAI](https://img.shields.io/badge/OpenAI-API-412991?style=for-the-badge&logo=openai)
![Qdrant](https://img.shields.io/badge/Qdrant-Vector_DB-ff5252?style=for-the-badge&logo=qdrant)
![Redis](https://img.shields.io/badge/Redis-Memory-DC382D?style=for-the-badge&logo=redis)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)

A production-ready, highly concurrent **AI Agent** built with Golang. This project implements an **Agentic RAG (Retrieval-Augmented Generation)** architecture capable of intelligently routing user queries, scraping web pages, reading PDFs, and maintaining conversational memory.

Unlike naive RAG systems, this assistant uses **OpenAI Function Calling (Tools)** to autonomously decide *when* to search its internal vector database and *when* to engage in general conversation.

## ✨ Key Features

- **🧠 Agentic Routing:** The AI autonomously decides whether to answer directly from its pre-trained knowledge or use the `search_local_documents` tool to fetch private data.
- **⚡ Concurrent Ingestion (The Power of Go):** Utilizes `Goroutines` and `Channels` to batch and process massive chunks of text (from web scraping or PDFs) concurrently, sending them to the Embedding API and Vector DB in milliseconds.
- **🌊 Real-time Streaming:** Implements Server-Sent Events (SSE) to stream AI responses token-by-token to the client, identical to the ChatGPT web experience.
- **🗂️ Vector Search:** Integrates **Qdrant** for lightning-fast semantic similarity search using `text-embedding-3-small` (1536 dimensions).
- **📝 Long-term Memory:** Uses **Redis** to maintain session-based conversation history, allowing the AI to remember context across multiple interactions.
- **🕷️ Web Scraping & PDF Parsing:** Built-in endpoints to ingest live web pages (using `colly`) and extract text from PDF files.

## 🏗️ Architecture

1. **Ingestion Flow:** Text (Web/PDF) ➔ Chunking ➔ Concurrent Goroutine Batches ➔ OpenAI Embeddings ➔ Qdrant.
2. **Chat Flow:** User Prompt ➔ Redis (Fetch History) ➔ Agent Evaluator (OpenAI Tool Call) ➔ (Optional: Qdrant Search) ➔ OpenAI Final Response ➔ SSE Stream to Client.

## 🚀 Getting Started

### Prerequisites
- [Go](https://golang.org/doc/install) (1.21 or newer)
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- An [OpenAI API Key](https://platform.openai.com/api-keys)

### 1. Clone the repository
```bash
git clone https://github.com/yourusername/go-rag-assistant.git
cd go-rag-assistant
```

### 2. Environment Variables
Create a `.env` file in the root directory and add your OpenAI API Key:
```env
PORT=8080
OPENAI_API_KEY=sk-your-openai-api-key-here
QDRANT_HOST=localhost
QDRANT_PORT=6334
REDIS_ADDR=localhost:6379
```

### 3. Start Infrastructure (Vector DB & Memory)
Run the provided Docker Compose file to spin up Qdrant and Redis:
```bash
docker compose up -d
```

### 4. Run the Go Server
```bash
go mod tidy
go run main.go
```

The server will start on `http://localhost:8080`.

## 🎮 Usage & Demo

### 1. Web UI (Chat & Streaming)
Simply open your browser and navigate to:
👉 http://localhost:8080
You will see a clean chat interface where you can interact with the AI in real-time.

### 2. Ingesting Data (API Examples)
**Scrape a Website**:
Feed the AI new knowledge by scraping an article:
```bash
curl -X POST http://localhost:8080/api/v1/scrape \
-H "Content-Type: application/json" \
-d '{"url": "https://en.wikipedia.org/wiki/Go_(programming_language)"}'
```

**Upload a PDF**:
```bash
curl -X POST http://localhost:8080/api/v1/upload \
-F "file=@/path/to/your/document.pdf"
```
After ingesting the data, go back to the Web UI and ask specific questions about the website or PDF. Watch the Agent use its "Tools" to search the database!

## 📂 Project Structure (Clean Architecture)
```text
.
├── docker-compose.yml       # Infrastructure setup (Qdrant, Redis)
├── main.go                  # Application entry point & DI
├── .env                     # Environment variables
├── static/
│   └── index.html           # Minimalist frontend with SSE support
└── internal/
    ├── agent/               # Agentic logic and Tool Calling definitions
    ├── api/                 # Gin HTTP Handlers & Routes
    ├── config/              # Configuration loader
    ├── ingest/              # Text chunking and Concurrent Processors
    ├── llm/                 # OpenAI API client wrappers
    ├── memory/              # Redis session management
    └── vectordb/            # Qdrant gRPC client wrapper
```

## 🤝 Contributing
Contributions are welcome! If you have ideas to improve the chunking strategy, add new tools (like a web-search tool using Tavily), or optimize the prompts, feel free to open a Pull Request.

## 📄 License
This project is licensed under the MIT License.

## ⚠️ Disclaimer & Copyright

**Copyright © 2026 [Afshin Talebi]. All rights reserved.**

- **Educational Purpose:** This project is developed primarily as a portfolio piece and for educational purposes to demonstrate modern backend architecture, concurrent processing in Go, and Agentic AI integration.
- **Third-Party Costs:** This application makes calls to the OpenAI API. Users are solely responsible for managing their own API keys and any associated costs or rate limits incurred while using this software.
- **"As-Is" Software:** The code is provided "as is", without warranty of any kind, express or implied. The author shall not be held liable for any damages, data loss, or issues arising from the use of this software in production environments. Please review and test thoroughly before deploying to any mission-critical systems.

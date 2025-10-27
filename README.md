# Vector Notes System

A Go-based system for syncing your notes to a vector database and querying them with AI assistance. The system consists of two main components:

- **vector-sync**: Monitors your notes directory and syncs changes to Pinecone vector database
- **note-gpt**: AI-powered query interface for your vectorized notes using Gemini

## Features

- 🔄 **Real-time sync**: Automatically detects file changes and updates the vector database
- 🔍 **Semantic search**: Query your notes using natural language
- 🤖 **AI assistance**: Get contextual answers from your notes using Google's Gemini
- 📁 **File watching**: Monitors `.md` files in your notes directory
- 🌲 **Tree structure**: Maintains directory structure for efficient syncing
- 💾 **Incremental updates**: Only processes changed files

## Prerequisites

- Go 1.21 or higher
- [Pinecone](https://www.pinecone.io/) account and API key
- [Google AI Studio](https://makersuite.google.com/app/apikey) API key for Gemini
- Local embedding server (Ollama recommended)

## Setup

### 1. Clone the Repository

```bash
git clone <your-repo-url>
cd browtomation
```

### 2. Install Dependencies

```bash
# Install vector-sync dependencies
cd vector-sync
go mod tidy

# Install note-gpt dependencies
cd ../note-gpt
go mod tidy
```

### 3. Setup Pinecone

1. Create a [Pinecone](https://www.pinecone.io/) account
2. Create a new index with the following specifications:
   - **Dimension**: 768 (for nomic-embed-text model)
   - **Metric**: Cosine
   - **Index Name**: `joyful-elm` (or update the code to match your index name)

### 4. Setup Local Embedding Server

Install and run Ollama with the nomic-embed-text model:

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull the embedding model
ollama pull nomic-embed-text

# Start Ollama server (runs on localhost:11434 by default)
ollama serve
```

### 5. Configure Environment Variables

Create `.env` files in both `vector-sync/` and `note-gpt/` directories:

**vector-sync/.env**:
```env
PINECONE_API_KEY=your_pinecone_api_key
PINECONE_HOST=https://your-index-host.pinecone.io
NOTES_DIR=/path/to/your/notes/directory
EMBEDDING_URL=http://localhost:11434/api/embed
```

**note-gpt/.env**:
```env
PINECONE_API_KEY=your_pinecone_api_key
PINECONE_HOST=https://your-index-host.pinecone.io
NOTES_DIR=/path/to/your/notes/directory
EMBEDDING_URL=http://localhost:11434/api/embed
GEMINI_API_KEY=your_gemini_api_key
```

Replace the placeholder values:
- `your_pinecone_api_key`: Your Pinecone API key
- `https://your-index-host.pinecone.io`: Your Pinecone index host URL
- `/path/to/your/notes/directory`: Absolute path to your notes directory
- `your_gemini_api_key`: Your Google AI Studio API key

## Usage

### Running Vector Sync

The vector-sync service monitors your notes directory and keeps the vector database synchronized:

```bash
cd vector-sync
go run main.go
```

This will:
- Build an initial tree structure of your notes
- Start watching for file changes
- Sync changes to Pinecone every 5 seconds
- Log all sync operations

### Running Note GPT

The note-gpt service provides an interactive query interface:

```bash
cd note-gpt
go run cmd/main.go
```

This starts an interactive session where you can:
- Ask questions about your notes
- Get AI-generated answers with file citations
- Maintain conversation context across queries

**Example interaction**:
```
> What are the cake ingredients?
Based on your notes from "Cake ingredients.md", you'll need flour, sugar, eggs, butter, and baking powder.

> How much flour?
According to the same file, you need 2 cups of all-purpose flour.
```

### Using VS Code

The repository includes VS Code launch configurations. You can:

1. Open the workspace in VS Code
2. Go to Run and Debug (Ctrl+Shift+D)
3. Select either "Launch Vector Sync" or "Launch note-gpt"
4. Press F5 to start debugging

## Architecture

### Vector Sync Flow
1. **File Watcher**: Monitors `.md` files using [`fsnotify`](vector-sync/internal/watcher.go)
2. **Tree Structure**: Maintains file hierarchy in [`Tree`](vector-sync/internal/tree.go)
3. **Diff Detection**: Compares client and server trees to find changes
4. **Vector Upsert**: Embeds content and stores in Pinecone via [`Vector`](vector-sync/pkg/vector.go)

### Note GPT Flow
1. **Query Processing**: Takes user input and vectorizes it
2. **Semantic Search**: Finds top 2 relevant notes from Pinecone
3. **Context Building**: Reads file contents and builds LLM context
4. **AI Response**: Generates response using Gemini with conversation history

## File Structure

```
├── vector-sync/           # Sync service
│   ├── internal/
│   │   ├── config.go     # Configuration management
│   │   ├── sync.go       # Main synchronization logic
│   │   ├── tree.go       # File tree operations
│   │   ├── node.go       # Tree node structure
│   │   ├── watcher.go    # File system watcher
│   │   └── utils.go      # Utility functions
│   ├── pkg/
│   │   ├── vector.go     # Pinecone integration
│   │   └── embedding.go  # Embedding API client
│   └── main.go           # Entry point
├── note-gpt/             # Query service
│   ├── cmd/
│   │   └── main.go       # CLI interface
│   ├── internal/
│   │   ├── app.go        # Main application logic
│   │   └── config.go     # Configuration management
│   └── pkg/
│       ├── vector.go     # Pinecone query operations
│       ├── embedding.go  # Embedding API client
│       └── gemini.go     # Gemini AI client
```

## Troubleshooting

### Common Issues

1. **Embedding server not running**:
   ```
   Error: connection refused to localhost:11434
   ```
   Solution: Make sure Ollama is running with `ollama serve`

2. **Pinecone authentication error**:
   ```
   Error: unauthorized
   ```
   Solution: Verify your `PINECONE_API_KEY` and `PINECONE_HOST` in `.env`

3. **No relevant files found**:
   - Ensure vector-sync has run and populated the database
   - Check that your notes directory contains `.md` files
   - Verify the notes directory path in `NOTES_DIR`

4. **Gemini API errors**:
   ```
   Error: failed to generate content
   ```
   Solution: Verify your `GEMINI_API_KEY` is valid and active

### Logging

Both services provide detailed logging:
- File operations and sync status
- Vector database operations
- API calls and responses
- Error details with context

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License.
package internal

import (
	"context"
	"fmt"
	"io/ioutil"
	"note-gpt/pkg"
	"path/filepath"
	"sync"

	"github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type App struct {
	Vector              *pkg.Vector
	LLM                 *pkg.GeminiClient
	conversationHistory []ConversationTurn
	mu                  sync.RWMutex
}

type ConversationTurn struct {
	Query    string
	Response string
	Context  []string // File contexts used for this turn
}

type FileContext struct {
	FilePath string
	Content  string
	Error    error
	Score    float32
}

func NewApp(vector *pkg.Vector, llm *pkg.GeminiClient) *App {
	return &App{
		Vector:              vector,
		LLM:                 llm,
		conversationHistory: make([]ConversationTurn, 0),
	}
}

func (a *App) HandleQuery(query string) (string, error) {
	ctx := context.Background()

	// Query vector database for top 2 matches
	matches, err := a.Vector.Query(ctx, []byte(query), 2)
	if err != nil {
		return "", fmt.Errorf("failed to query vector database: %w", err)
	}

	// Read files concurrently
	contexts := a.readFilesConcurrently(matches)

	// Combine contexts for LLM
	if len(contexts) == 0 {
		return "No relevant files found for the query.", nil
	}

	systemPrompt := `You are a command-line LLM assistant. You are provided context from the user's local notes, which are synced every 30 seconds with a vector database. 
For each query, only the top 2 relevant notes are retrieved and passed to you. 
Based on the query and the provided context, answer concisely. 
If the context does not contain an answer, say "I don't know." 
Always cite the file names from which your information is taken, when relevant.
You can refer to previous conversation turns when answering follow-up questions.`

	// Build conversation history for context
	conversationContext := a.buildConversationContext()

	combinedContext := fmt.Sprintf("<SYSTEM_PROMPT>\n%s\n</SYSTEM_PROMPT>\n\n%s<CURRENT_QUERY>\n%s\n</CURRENT_QUERY>\n\n<CURRENT_CONTEXT>\n%s\n</CURRENT_CONTEXT>\n\n",
		systemPrompt, conversationContext, query, joinContexts(contexts))

	response, err := a.LLM.GenerateResponse(combinedContext)
	if err != nil {
		return "", fmt.Errorf("failed to generate LLM response: %w", err)
	}

	// Store this conversation turn
	a.addConversationTurn(query, response, contexts)

	fmt.Printf("Combined Context for LLM:\n%s\n", combinedContext)
	return response, nil
}

func (a *App) buildConversationContext() string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.conversationHistory) == 0 {
		return ""
	}

	// Include last 3 conversation turns for context (adjust as needed)
	maxTurns := 3
	startIdx := 0
	if len(a.conversationHistory) > maxTurns {
		startIdx = len(a.conversationHistory) - maxTurns
	}

	context := "<CONVERSATION_HISTORY>\n"
	for i := startIdx; i < len(a.conversationHistory); i++ {
		turn := a.conversationHistory[i]
		context += fmt.Sprintf("Previous Query: %s\nPrevious Response: %s\n\n", turn.Query, turn.Response)
	}
	context += "</CONVERSATION_HISTORY>\n\n"

	return context
}

func (a *App) addConversationTurn(query, response string, contexts []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	turn := ConversationTurn{
		Query:    query,
		Response: response,
		Context:  contexts,
	}

	a.conversationHistory = append(a.conversationHistory, turn)

	// Keep only last 5 turns to prevent context from growing too large
	maxHistory := 5
	if len(a.conversationHistory) > maxHistory {
		a.conversationHistory = a.conversationHistory[len(a.conversationHistory)-maxHistory:]
	}
}

// ClearHistory clears the conversation history
func (a *App) ClearHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.conversationHistory = make([]ConversationTurn, 0)
}

// GetConversationHistory returns a copy of the conversation history
func (a *App) GetConversationHistory() []ConversationTurn {
	a.mu.RLock()
	defer a.mu.RUnlock()

	history := make([]ConversationTurn, len(a.conversationHistory))
	copy(history, a.conversationHistory)
	return history
}

func (a *App) readFilesConcurrently(matches []*pinecone.ScoredVector) []string {
	var wg sync.WaitGroup
	resultChan := make(chan FileContext, len(matches))

	// Launch goroutines for each file
	for _, match := range matches {
		if match.Vector.Metadata == nil {
			continue
		}

		metadata := match.Vector.Metadata.AsMap()
		filePathVal := metadata["filepath"]
		filePath, ok := filePathVal.(string)
		if !ok {
			continue
		}

		wg.Add(1)
		go func(path string, score float32) {
			defer wg.Done()

			content, err := a.readFile(path)
			resultChan <- FileContext{
				FilePath: path,
				Content:  content,
				Error:    err,
				Score:    score,
			}
		}(filePath, match.Score)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and sort by relevance score
	var fileContexts []FileContext
	for result := range resultChan {
		if result.Error != nil {
			fmt.Printf("Warning: failed to read file %s: %v\n", result.FilePath, result.Error)
			continue
		}
		fileContexts = append(fileContexts, result)
	}

	// Sort by score (higher scores first - more relevant)
	for i := 0; i < len(fileContexts)-1; i++ {
		for j := i + 1; j < len(fileContexts); j++ {
			if fileContexts[i].Score < fileContexts[j].Score {
				fileContexts[i], fileContexts[j] = fileContexts[j], fileContexts[i]
			}
		}
	}

	// Convert to string contexts
	var contexts []string
	for _, fc := range fileContexts {
		contexts = append(contexts, fmt.Sprintf("File: %s\nContent:\n%s\n", fc.FilePath, fc.Content))
	}

	return contexts
}

func (a *App) readFile(filePath string) (string, error) {
	cleanPath := filepath.Clean(filePath)
	content, err := ioutil.ReadFile(cleanPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func joinContexts(contexts []string) string {
	result := ""
	for i, context := range contexts {
		if i > 0 {
			result += "\n---\n\n"
		}
		result += context
	}
	return result
}

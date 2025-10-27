package pkg

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Embedding struct {
	httpClient   *http.Client
	embeddingUrl string
}

type EmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func NewEmbedding(embeddingUrl string) *Embedding {
	client := &http.Client{}
	return &Embedding{
		httpClient:   client,
		embeddingUrl: embeddingUrl,
	}
}

func (e *Embedding) Vectorize(text string) ([]float32, error) {
	body, _ := json.Marshal(EmbedRequest{
		Model: "nomic-embed-text",
		Input: text,
	})
	resp, err := e.httpClient.Post(e.embeddingUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Embeddings[0], nil
}

package pkg

import (
	"context"
	"log"

	"github.com/pinecone-io/go-pinecone/v4/pinecone"
)

type Vector struct {
	db       *pinecone.IndexConnection
	embedder *Embedding
}

func NewVector(apiKey, host, indexName string, embeddingUrl string) (*Vector, error) {
	client, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	index, err := client.Index(pinecone.NewIndexConnParams{
		Host: host,
	})
	if err != nil {
		return nil, err
	}
	embedder := NewEmbedding(embeddingUrl)
	return &Vector{db: index, embedder: embedder}, nil
}

func (v *Vector) Query(ctx context.Context, queryText []byte, topK int) ([]*pinecone.ScoredVector, error) {
	log.Printf("Vectorizing query text: %s", string(queryText))
	vectorizedText, err := v.embedder.Vectorize(string(queryText))
	if err != nil {
		return nil, err
	}
	log.Printf("Querying vector database with vector: %f", vectorizedText)
	query := pinecone.QueryByVectorValuesRequest{
		TopK:            uint32(topK),
		IncludeValues:   false,
		IncludeMetadata: true,
		Vector:          vectorizedText,
	}
	response, err := v.db.QueryByVectorValues(ctx, &query)
	if err != nil {
		return nil, err
	}
	return response.Matches, nil
}

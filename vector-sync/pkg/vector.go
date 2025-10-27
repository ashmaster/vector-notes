package pkg

import (
	"context"
	"log"

	"github.com/pinecone-io/go-pinecone/v4/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
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

func (v *Vector) vectorizeText(text []byte, vectorizedText *[]float32) error {
	var err error
	*vectorizedText, err = v.embedder.Vectorize(string(text))
	return err
}

func (v *Vector) Upsert(ctx context.Context, id string, text []byte, filepath string, lastmodified string) error {
	log.Printf("Vectorizing text for file: %s", filepath)
	var vectorizedText []float32
	err := v.vectorizeText(text, &vectorizedText)
	if err != nil {
		return err
	}
	metadataMap := map[string]interface{}{
		"filepath": filepath,
		"modified": lastmodified,
	}

	metadata, err := structpb.NewStruct(metadataMap)
	if err != nil {
		return err
	}
	log.Printf("Upserting vector for file: %s", filepath)
	record := pinecone.Vector{
		Id:       id,
		Values:   &vectorizedText,
		Metadata: metadata,
	}
	_, err = v.db.UpsertVectors(ctx, []*pinecone.Vector{&record})
	if err != nil {
		return err
	}
	log.Printf("Upserted vector for file: %s", filepath)
	return nil
}

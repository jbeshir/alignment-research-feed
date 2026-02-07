package datasources

import "context"

// Embedder embeds text into a vector for similarity search.
type Embedder interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
}

// NullEmbedder is a null implementation of Embedder.
type NullEmbedder struct{}

var _ Embedder = NullEmbedder{}

func (NullEmbedder) EmbedText(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

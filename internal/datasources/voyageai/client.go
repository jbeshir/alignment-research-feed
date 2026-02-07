package voyageai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
)

var _ datasources.Embedder = (*Client)(nil)

// Client embeds text using the VoyageAI contextual embeddings API.
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient creates a new VoyageAI client.
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:     apiKey,
		model:      model,
		httpClient: http.DefaultClient,
	}
}

type embeddingRequest struct {
	Inputs          [][]string `json:"inputs"`
	Model           string     `json:"model"`
	InputType       string     `json:"input_type"`
	OutputDimension int        `json:"output_dimension"`
}

type embeddingResponse struct {
	Data []struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	} `json:"data"`
}

func (c *Client) EmbedText(ctx context.Context, text string) ([]float32, error) {
	reqBody := embeddingRequest{
		Inputs:          [][]string{{text}},
		Model:           c.model,
		InputType:       "query",
		OutputDimension: 1024,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.voyageai.com/v1/contextualizedembeddings",
		bytes.NewReader(jsonBody),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("VoyageAI API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(result.Data) == 0 || len(result.Data[0].Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return result.Data[0].Data[0].Embedding, nil
}

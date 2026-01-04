package pinecone

import (
	"context"
	"fmt"
	"strings"

	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/pinecone-io/go-pinecone/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ datasources.SimilarArticleLister = (*Client)(nil)

type Client struct {
	pinecone *pinecone.Client
	index    *pinecone.Index
}

func NewClient(
	ctx context.Context,
	apiKey string,
) (*Client, error) {
	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey:     apiKey,
		Headers:    nil,
		Host:       "",
		RestClient: nil,
		SourceTag:  "",
	})
	if err != nil {
		return nil, fmt.Errorf("creating pinecone client: %w", err)
	}

	idx, err := pc.DescribeIndex(ctx, "alignment-research-dataset")
	if err != nil {
		return nil, fmt.Errorf("retrieving pinecone index metadata for dataset: %w", err)
	}

	return &Client{
		pinecone: pc,
		index:    idx,
	}, nil
}

func (c *Client) ListSimilarArticles(
	ctx context.Context,
	hashIDs []string,
	limit int,
) ([]domain.SimilarArticle, error) {
	if limit > 10000 {
		return nil, fmt.Errorf("limit value too high [%d]", limit)
	}
	if len(hashIDs) == 0 {
		return nil, nil
	}

	idxConn, err := c.pinecone.Index(pinecone.NewIndexConnParams{
		Host:      c.index.Host,
		Namespace: "normal",
	})
	if err != nil {
		return nil, fmt.Errorf("creating pinecone index connection: %w", err)
	}
	defer func() {
		if closeErr := idxConn.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	// Fetch vectors for each hash ID and average them
	var allVectors [][]float32
	for _, hashID := range hashIDs {
		vector, err := c.getBaseSearchVector(ctx, idxConn, hashID)
		if err != nil {
			// Skip articles that don't have vectors
			continue
		}
		allVectors = append(allVectors, vector)
	}

	if len(allVectors) == 0 {
		return nil, nil
	}

	searchVector := averageVectors(allVectors)

	return c.findSimilarArticles(ctx, idxConn, hashIDs, searchVector, limit)
}

func (c *Client) getBaseSearchVector(
	ctx context.Context,
	idxConn *pinecone.IndexConnection,
	hashID string,
) ([]float32, error) {
	baseVectorPrefix := hashID + "_"
	baseVectorLimit := uint32(20)
	baseVectorIDsResp, err := idxConn.ListVectors(ctx, &pinecone.ListVectorsRequest{
		Prefix:          &baseVectorPrefix,
		Limit:           &baseVectorLimit,
		PaginationToken: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("listing vector IDs for base article [%s]: %w", hashID, err)
	}
	if len(baseVectorIDsResp.VectorIds) == 0 {
		return nil, fmt.Errorf("no vectors IDs found for article [%s]", hashID)
	}

	var baseVectorIDs []string
	for _, id := range baseVectorIDsResp.VectorIds {
		baseVectorIDs = append(baseVectorIDs, *id)
	}

	baseVectorsResp, err := idxConn.FetchVectors(ctx, baseVectorIDs)
	if err != nil {
		return nil, fmt.Errorf("fetching vectors for base article [%s]: %w", hashID, err)
	}

	return averagePineconeVectors(baseVectorsResp.Vectors), nil
}

func (c *Client) findSimilarArticles(
	ctx context.Context,
	idxConn *pinecone.IndexConnection,
	excludeHashIDs []string,
	searchVector []float32,
	limit int,
) ([]domain.SimilarArticle, error) {
	var results []domain.SimilarArticle

	for len(results) < limit {
		foundResult, err := c.searchBatch(ctx, idxConn, excludeHashIDs, searchVector, &results, limit)
		if err != nil {
			return nil, err
		}
		if !foundResult {
			break // No more results to find, stop even though we're not at limit
		}
	}

	return results, nil
}

func (c *Client) searchBatch(
	ctx context.Context,
	idxConn *pinecone.IndexConnection,
	excludeHashIDs []string,
	searchVector []float32,
	results *[]domain.SimilarArticle,
	limit int,
) (bool, error) {
	filter, err := c.createExistingResultsExclusionFilter(excludeHashIDs, *results)
	if err != nil {
		return false, err
	}

	resp, err := idxConn.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
		Vector:          searchVector,
		TopK:            10,
		MetadataFilter:  filter,
		IncludeValues:   false,
		IncludeMetadata: false,
		SparseValues:    nil,
	})
	if err != nil {
		return false, fmt.Errorf("querying for similar vectors: %w", err)
	}

	return c.processSearchResults(resp, results, limit)
}

func (c *Client) createExistingResultsExclusionFilter(
	excludeHashIDs []string,
	results []domain.SimilarArticle,
) (*pinecone.MetadataFilter, error) {
	var filterExistingIDs []any
	for _, id := range excludeHashIDs {
		filterExistingIDs = append(filterExistingIDs, id)
	}
	for _, result := range results {
		filterExistingIDs = append(filterExistingIDs, result.HashID)
	}

	metadataMap := map[string]any{
		"hash_id": map[string]any{
			"$nin": filterExistingIDs,
		},
	}

	filter, err := structpb.NewStruct(metadataMap)
	if err != nil {
		return nil, fmt.Errorf("creating metadata filter map: %w", err)
	}
	return filter, nil
}

func (c *Client) processSearchResults(
	resp *pinecone.QueryVectorsResponse,
	results *[]domain.SimilarArticle,
	limit int,
) (bool, error) {
	foundResult := false

	for _, scoredVector := range resp.Matches {
		matchHashID, err := c.extractHashIDFromVector(scoredVector.Vector.Id)
		if err != nil {
			return false, err
		}

		if c.isDuplicate(matchHashID, *results) {
			continue
		}

		foundResult = true
		if len(*results) < limit {
			*results = append(*results, domain.SimilarArticle{
				HashID: matchHashID,
				Score:  float64(scoredVector.Score),
			})
		}
	}

	return foundResult, nil
}

func (c *Client) extractHashIDFromVector(vectorID string) (string, error) {
	vectorIDParts := strings.Split(vectorID, "_")
	if len(vectorIDParts) < 2 {
		return "", fmt.Errorf("unexpected pinecone vector ID format [%s]", vectorID)
	}
	return vectorIDParts[0], nil
}

func (c *Client) isDuplicate(hashID string, results []domain.SimilarArticle) bool {
	for _, result := range results {
		if result.HashID == hashID {
			return true
		}
	}
	return false
}

func averagePineconeVectors(vectors map[string]*pinecone.Vector) []float32 {
	var values [][]float32
	for _, vector := range vectors {
		values = append(values, vector.Values)
	}
	return averageVectors(values)
}

func averageVectors(vectors [][]float32) []float32 {
	if len(vectors) == 0 {
		return nil
	}

	result := make([]float32, len(vectors[0]))
	for _, vector := range vectors {
		for i, v := range vector {
			result[i] += v
		}
	}

	for i := range result {
		result[i] /= float32(len(vectors))
	}

	return result
}

package pinecone

import (
	"context"
	"fmt"
	"github.com/jbeshir/alignment-research-feed/internal/datasources"
	"github.com/jbeshir/alignment-research-feed/internal/domain"
	"github.com/pinecone-io/go-pinecone/pinecone"
	"google.golang.org/protobuf/types/known/structpb"
	"strings"
)

var _ datasources.SimilarityRepository = (*Client)(nil)

type Client struct {
	pinecone      *pinecone.Client
	idxConnection *pinecone.IndexConnection
}

func NewClient(
	ctx context.Context,
	apiKey string,
) (*Client, error) {
	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("creating pinecone client: %w", err)
	}

	idx, err := pc.DescribeIndex(ctx, "alignment-research-dataset")
	if err != nil {
		return nil, fmt.Errorf("retrieving pinecone index metadata for dataset: %w", err)
	}

	idxConn, err := pc.Index(pinecone.NewIndexConnParams{
		Host: idx.Host,
	})
	if err != nil {
		return nil, fmt.Errorf("creating pinecone index connection: %w", err)
	}

	return &Client{
		pinecone:      pc,
		idxConnection: idxConn,
	}, nil
}

func (c *Client) ListSimilarArticles(ctx context.Context, hashID string, limit int) ([]domain.SimilarArticle, error) {
	if limit > 10000 {
		return nil, fmt.Errorf("limit value too high [%d]", limit)
	}

	baseVectorPrefix := hashID + "_"
	baseVectorLimit := uint32(20)
	baseVectorIDsResp, err := c.idxConnection.ListVectors(ctx, &pinecone.ListVectorsRequest{
		Prefix:          &baseVectorPrefix,
		Limit:           &baseVectorLimit,
		PaginationToken: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("listing vector IDs for base article: %w", err)
	}

	var baseVectorIDs []string
	for _, id := range baseVectorIDsResp.VectorIds {
		baseVectorIDs = append(baseVectorIDs, *id)
	}
	baseVectorsResp, err := c.idxConnection.FetchVectors(ctx, baseVectorIDs)
	if err != nil {
		return nil, fmt.Errorf("fetching vectors for base article: %w", err)
	}

	searchVector := averageVectorValues(baseVectorsResp.Vectors)

	var results []domain.SimilarArticle
	for len(results) < limit {
		var filter *pinecone.MetadataFilter
		if len(results) > 1 {
			filterExistingIDs := []string{hashID}
			for _, result := range results {
				filterExistingIDs = append(filterExistingIDs, result.HashID)
			}

			metadataMap := map[string]any{
				"hash_id": map[string]any{
					"$nin": filterExistingIDs,
				},
			}

			filter, err = structpb.NewStruct(metadataMap)
			if err != nil {
				return nil, fmt.Errorf("creating metadata filter map: %w", err)
			}
		}

		resp, err := c.idxConnection.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
			Vector:          searchVector,
			TopK:            10,
			MetadataFilter:  filter,
			IncludeValues:   false,
			IncludeMetadata: false,
			SparseValues:    nil,
		})
		if err != nil {
			return nil, fmt.Errorf("querying for similar vectors: %w", err)
		}

		foundResult := false
		for _, scoredVector := range resp.Matches {
			vectorID := scoredVector.Vector.Id
			vectorIDParts := strings.Split(vectorID, "_")
			if len(vectorIDParts) < 2 {
				return nil, fmt.Errorf("unexpected pinecone vector ID format [%s]", vectorID)
			}
			matchHashID := vectorIDParts[0]

			duplicate := false
			for _, result := range results {
				if result.HashID == matchHashID {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}

			foundResult = true
			if len(results) < limit {
				results = append(results, domain.SimilarArticle{
					HashID: matchHashID,
					Score:  float64(scoredVector.Score),
				})
			}
		}

		// If we can't find any more results, stop looping
		if !foundResult {
			break
		}
	}

	return results, nil
}

func averageVectorValues(vectors map[string]*pinecone.Vector) []float32 {
	var values []float32
	for _, vector := range vectors {
		if values == nil {
			values = append(values, vector.Values...)
			continue
		}

		for i, v := range values {
			values[i] += v
		}
	}

	for i := range values {
		values[i] /= float32(len(vectors))
	}

	return values
}
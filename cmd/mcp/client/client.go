// Package client provides an HTTP client for the Alignment Research Feed API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Article represents an article from the alignment research feed.
type Article struct {
	HashID      string    `json:"hash_id"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	TextStart   string    `json:"text_start"`
	Authors     string    `json:"authors"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`

	Summary     string   `json:"summary,omitempty"`
	KeyPoints   []string `json:"key_points,omitempty"`
	Implication string   `json:"implication,omitempty"`
	Category    string   `json:"category,omitempty"`

	HaveRead   *bool `json:"have_read,omitempty"`
	ThumbsUp   *bool `json:"thumbs_up,omitempty"`
	ThumbsDown *bool `json:"thumbs_down,omitempty"`
}

// ArticlesResponse represents the paginated response for article lists.
type ArticlesResponse struct {
	Data     []Article `json:"data"`
	Metadata struct{}  `json:"metadata"`
}

// SearchFilters contains search parameters for listing articles.
type SearchFilters struct {
	Query           string
	Sources         []string
	ExcludeSources  []string
	PublishedAfter  *time.Time
	PublishedBefore *time.Time
	Limit           int
	Page            int
	Sort            string
	Category        string
}

// Client is an HTTP client for the Alignment Research Feed API.
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// NewClient creates a new API client.
func NewClient(baseURL, apiToken string) *Client {
	return &Client{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string) (*http.Response, error) {
	return c.doRequestWithBody(ctx, method, path, nil)
}

func (c *Client) doRequestWithBody(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	return resp, nil
}

func (c *Client) handleResponse(resp *http.Response, result interface{}) error {
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

func (f SearchFilters) queryParams() url.Values {
	params := url.Values{}

	if f.Query != "" {
		params.Set("filter_title_fulltext", f.Query)
	}
	if len(f.Sources) > 0 {
		params.Set("filter_sources_allowlist", strings.Join(f.Sources, ","))
	}
	if len(f.ExcludeSources) > 0 {
		params.Set("filter_sources_blocklist", strings.Join(f.ExcludeSources, ","))
	}
	if f.PublishedAfter != nil {
		params.Set("filter_published_after", f.PublishedAfter.Format(time.RFC3339))
	}
	if f.PublishedBefore != nil {
		params.Set("filter_published_before", f.PublishedBefore.Format(time.RFC3339))
	}
	if f.Limit > 0 {
		params.Set("page_size", strconv.Itoa(f.Limit))
	}
	if f.Page > 0 {
		params.Set("page", strconv.Itoa(f.Page))
	}
	if f.Category != "" {
		params.Set("filter_category", f.Category)
	}
	if f.Sort != "" {
		params.Set("sort", f.Sort)
	} else {
		params.Set("sort", "published_at_desc")
	}

	return params
}

// SearchArticles searches for articles with the given filters.
func (c *Client) SearchArticles(ctx context.Context, filters SearchFilters) ([]Article, error) {
	params := filters.queryParams()

	path := "/v1/articles"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result ArticlesResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetArticle retrieves a single article by its hash ID.
func (c *Client) GetArticle(ctx context.Context, articleID string) (*Article, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/v1/articles/"+url.PathEscape(articleID))
	if err != nil {
		return nil, err
	}

	var article Article
	if err := c.handleResponse(resp, &article); err != nil {
		return nil, err
	}

	return &article, nil
}

// GetSimilarArticles finds articles similar to the given article.
func (c *Client) GetSimilarArticles(ctx context.Context, articleID string, limit int) ([]Article, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	path := "/v1/articles/" + url.PathEscape(articleID) + "/similar"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result ArticlesResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// SemanticSearch finds articles semantically similar to the given text.
func (c *Client) SemanticSearch(ctx context.Context, text string, limit int) ([]Article, error) {
	reqBody := struct {
		Text  string `json:"text"`
		Limit int    `json:"limit"`
	}{
		Text:  text,
		Limit: limit,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	resp, err := c.doRequestWithBody(ctx, http.MethodPost, "/v1/articles/semantic-search", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	var result ArticlesResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetRecommendations retrieves personalized article recommendations.
func (c *Client) GetRecommendations(ctx context.Context, limit int) ([]Article, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	path := "/v1/articles/recommended"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result ArticlesResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// RateArticle sets the thumbs up or thumbs down rating for an article.
func (c *Client) RateArticle(ctx context.Context, articleID string, thumbsUp, thumbsDown bool) error {
	// Set thumbs up
	upPath := fmt.Sprintf("/v1/articles/%s/thumbs_up/%t", url.PathEscape(articleID), thumbsUp)
	resp, err := c.doRequest(ctx, http.MethodPost, upPath)
	if err != nil {
		return err
	}
	if err := c.handleResponse(resp, nil); err != nil {
		return fmt.Errorf("setting thumbs_up: %w", err)
	}

	// Set thumbs down
	downPath := fmt.Sprintf("/v1/articles/%s/thumbs_down/%t", url.PathEscape(articleID), thumbsDown)
	resp, err = c.doRequest(ctx, http.MethodPost, downPath)
	if err != nil {
		return err
	}
	if err := c.handleResponse(resp, nil); err != nil {
		return fmt.Errorf("setting thumbs_down: %w", err)
	}

	return nil
}

// MarkRead marks an article as read or unread.
func (c *Client) MarkRead(ctx context.Context, articleID string, read bool) error {
	path := fmt.Sprintf("/v1/articles/%s/read/%t", url.PathEscape(articleID), read)
	resp, err := c.doRequest(ctx, http.MethodPost, path)
	if err != nil {
		return err
	}
	return c.handleResponse(resp, nil)
}

// ListLiked retrieves articles the user has liked (thumbs up).
func (c *Client) ListLiked(ctx context.Context, page, pageSize int) ([]Article, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.Itoa(pageSize))
	}

	path := "/v1/articles/liked"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result ArticlesResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// ListDisliked retrieves articles the user has disliked (thumbs down).
func (c *Client) ListDisliked(ctx context.Context, page, pageSize int) ([]Article, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.Itoa(pageSize))
	}

	path := "/v1/articles/disliked"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result ArticlesResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// ListUnreviewed retrieves articles the user hasn't reviewed yet.
func (c *Client) ListUnreviewed(ctx context.Context, page, pageSize int) ([]Article, error) {
	params := url.Values{}
	if page > 0 {
		params.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		params.Set("page_size", strconv.Itoa(pageSize))
	}

	path := "/v1/articles/unreviewed"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path)
	if err != nil {
		return nil, err
	}

	var result ArticlesResponse
	if err := c.handleResponse(resp, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

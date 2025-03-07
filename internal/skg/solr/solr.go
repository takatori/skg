package solr

import (
	"context"
	"fmt"

	"github.com/takatori/skg/internal"
	"github.com/takatori/skg/internal/infra"
	"github.com/takatori/skg/internal/skg"
)

type SolrSemanticKnowledgeGraph struct {
	config     *internal.Config
	httpClient *infra.HttpClient
}

// NewSolrSemanticKnowledgeGraph creates a new SolrSemanticKnowledgeGraph with the given config
// and initializes the HTTP client
func NewSolrSemanticKnowledgeGraph(config *internal.Config) *SolrSemanticKnowledgeGraph {
	return &SolrSemanticKnowledgeGraph{
		config:     config,
		httpClient: infra.NewHttpClient(),
	}
}

// NewSolrSemanticKnowledgeGraphWithClient creates a new SolrSemanticKnowledgeGraph with the given config
// and HTTP client
func NewSolrSemanticKnowledgeGraphWithClient(config *internal.Config, httpClient *infra.HttpClient) *SolrSemanticKnowledgeGraph {
	return &SolrSemanticKnowledgeGraph{
		config:     config,
		httpClient: httpClient,
	}
}

func (s *SolrSemanticKnowledgeGraph) Traverse(ctx context.Context, q [][]skg.Query, collection string) (map[string]skg.Traversal, error) {
	// Use default collection if none provided
	if collection == "" {
		collection = "products"
	}

	reqBody := transformRequest(q)
	url := fmt.Sprintf("%s/%s/query", s.config.SolrUrl, collection)

	// Create a response map to hold the Solr response
	var solrResp map[string]interface{}

	// Use the HTTP client to make the request
	err := s.httpClient.Post(
		ctx,
		infra.PostRequest{
			Request: infra.Request{
				Url: url,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
			},
			Entity: reqBody,
		},
		&solrResp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send post request: %w", err)
	}
	node := solrResp["facets"].(map[string]interface{})
	converter := ResponseConverter{
		RequestParams: reqBody["params"].(map[string]interface{}),
	}
	return converter.transformResponseFacet(node), nil
}

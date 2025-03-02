package solr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/takatori/skg/internal"
	"github.com/takatori/skg/internal/skg"
)

type SolrSemanticKnowledgeGraph struct {
	config *internal.Config
}

func NewSolrSemanticKnowledgeGraph(config *internal.Config) *SolrSemanticKnowledgeGraph {
	return &SolrSemanticKnowledgeGraph{
		config: config,
	}
}

func (s *SolrSemanticKnowledgeGraph) Traverse(q [][]skg.Query, collection string) (map[string]skg.Traversal, error) {
	// Get Solr URL from config
	solrURL := s.config.SolrUrl

	// Use default collection if none provided
	if collection == "" {
		collection = "products"
	}

	reqBody := transformRequest(q)

	payload, err := json.Marshal(reqBody)
	fmt.Println(string(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/%s/query", solrURL, collection)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to send post request: %w", err)

	}
	defer resp.Body.Close()

	var solrResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&solrResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return transformResponseFacet(solrResp["facets"].(map[string]interface{}), reqBody["params"].(map[string]interface{})), nil

}

package handler

import (
	"net/http"
	"sort"

	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"github.com/takatori/skg/internal"
	"github.com/takatori/skg/internal/infra"
	"github.com/takatori/skg/internal/skg"
	"github.com/takatori/skg/internal/skg/solr"
)

// RelatedTermsParams defines the parameters for the related terms API
type RelatedTermsParams struct {
	Keyword    string `json:"keyword" validate:"required"`
	Collection string `json:"collection" validate:"required"`
}

// RelatedTerm represents a single term related to the input keyword
type RelatedTerm struct {
	Term        string  `json:"term"`
	Relatedness float64 `json:"relatedness"`
}

type CalcRelatednessParams struct {
	Keyword    string `json:"keyword" validate:"required"`
	Document   string `json:"document" validate:"required"`
	Collection string `json:"collection" validate:"required"`
}

// RelatedTermsHandler handles requests for related terms
type RelatedTermsHandler struct {
	config     *internal.Config
	httpClient *infra.HttpClient
}

// NewRelatedTermsHandlerWithClient creates a new RelatedTermsHandler with the given config and HTTP client
func NewRelatedTermsHandlerWithClient(config *internal.Config, httpClient *infra.HttpClient) *RelatedTermsHandler {
	return &RelatedTermsHandler{
		config:     config,
		httpClient: httpClient,
	}
}

// RelatedTermsEndpoint returns an Echo handler function for the related terms endpoint
func (h *RelatedTermsHandler) RelatedTermsEndpoint() func(echo.Context) error {
	return func(c echo.Context) error {
		// Parse and validate request parameters
		params, err := parseParams(c)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Build queries for the semantic knowledge graph
		queries := buildQueries(params.Keyword)

		// Query the semantic knowledge graph
		skgInstance := solr.NewSolrSemanticKnowledgeGraphWithClient(h.config, h.httpClient)
		result, err := skgInstance.Traverse(c.Request().Context(), queries, params.Collection)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Process results into related terms
		relatedTerms := extractRelatedTerms(result)

		return c.JSON(http.StatusOK, relatedTerms)
	}
}

func (h *RelatedTermsHandler) CalcRelatedness() func(echo.Context) error {

	return func(c echo.Context) error {
		params, err := parseCalcParams(c)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		// Use Kagome tokenizer with IPA dictionary for morphological analysis
		t, err := tokenizer.New(ipa.Dict())
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to initialize tokenizer"})
		}
		tokens := t.Tokenize(params.Document)

		// Extract meaningful words (nouns, verbs, adjectives, etc.)
		var phrases []string
		for _, token := range tokens {
			// Skip punctuation, symbols, and other non-content tokens
			if token.Class == tokenizer.DUMMY {
				continue
			}

			// Get the base form of the word
			features := token.Features()
			if len(features) > 0 && features[0] != "名詞" {
				continue
			}

			if len(features) > 6 && features[6] != "*" {
				// Use base form if available
				phrases = append(phrases, features[6])
			} else {
				// Otherwise use the surface form
				phrases = append(phrases, token.Surface)
			}
		}

		queries := [][]skg.Query{
			{
				{
					Field: "text",
					Values: []string{
						params.Keyword,
					},
				},
			},
			{
				{
					Field:  "text",
					Values: phrases,
				},
			},
		}

		skgInstance := solr.NewSolrSemanticKnowledgeGraphWithClient(h.config, h.httpClient)
		result, err := skgInstance.Traverse(c.Request().Context(), queries, params.Collection)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Convert result to RelatedTerm objects
		response := extractRelatedTermsForCalc(result)

		// Sort by relatedness in descending order
		sort.Slice(response, func(i, j int) bool {
			return response[i].Relatedness > response[j].Relatedness
		})

		return c.JSON(http.StatusOK, response)
	}
}

// For backward compatibility
func NewRelatedTermsHandler(config *internal.Config) func(echo.Context) error {
	handler := NewRelatedTermsHandlerWithClient(config, infra.NewHttpClient())
	return handler.RelatedTermsEndpoint()
}

// parseParams extracts and validates the request parameters
func parseParams(c echo.Context) (RelatedTermsParams, error) {
	var params RelatedTermsParams
	if err := c.Bind(&params); err != nil {
		return params, err
	}
	return params, nil
}

func parseCalcParams(c echo.Context) (CalcRelatednessParams, error) {
	var params CalcRelatednessParams
	if err := c.Bind(&params); err != nil {
		return params, err
	}
	return params, nil
}

// buildQueries constructs the query structure for the semantic knowledge graph
func buildQueries(keyword string) [][]skg.Query {
	return [][]skg.Query{
		{
			{
				Field: "text",
				Values: []string{
					keyword,
				},
			},
		},
		{
			{
				Field:         "text",
				MinOccurrence: lo.ToPtr(2),
				Limit:         lo.ToPtr(8),
			},
		},
	}
}

// extractRelatedTerms processes the SKG result into a list of related terms
func extractRelatedTerms(result map[string]skg.Traversal) []RelatedTerm {
	var relatedTerms []RelatedTerm

	for _, item := range result {
		// Skip if there are no values
		if len(item.Values) == 0 {
			continue
		}

		// Process each traversal in the first value
		for _, traversal := range item.Values[0].Traversals {
			for _, value := range traversal.Values {
				relatedTerms = append(relatedTerms, RelatedTerm{
					Term:        value.Key,
					Relatedness: value.Relatedness,
				})
			}
		}
	}

	return relatedTerms
}

// extractRelatedTermsForCalc processes the SKG result into a list of related terms for the CalcRelatedness function
func extractRelatedTermsForCalc(result map[string]skg.Traversal) []RelatedTerm {
	var relatedTerms []RelatedTerm

	for _, item := range result {
		// Skip if there are no values
		if len(item.Values) == 0 {
			continue
		}

		// Process each traversal in the first value
		for _, traversal := range item.Values[0].Traversals {
			for _, value := range traversal.Values {
				relatedTerms = append(relatedTerms, RelatedTerm{
					Term:        value.Key,
					Relatedness: value.Relatedness,
				})
			}
		}
	}

	return relatedTerms
}

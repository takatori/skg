package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func NewSkgHandler() func(echo.Context) error {
	return func(c echo.Context) error {
		return c.String(
			http.StatusOK, "Hello, World! Skg",
		)
	}
}

type RelatedTermsParams struct {
	Keyword    string `json:"keyword" validate:"required"`
	Collection string `json:"collection" validate:"required"`
}

type RelatedTerm struct {
	Term        string `json:"term"`
	Relatedness string `json:"relatedness"`
}

// NewRelatedTermsHandler queries Solr and returns formatted related terms.
func NewRelatedTermsHandler() func(echo.Context) error {
	return func(c echo.Context) error {
		// Define the Solr URL and collection.
		solrURL := "http://solr:8983/solr"

		var params RelatedTermsParams
		if err := c.Bind(&params); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}

		// Build the request payload.
		reqBody := map[string]interface{}{
			"params": map[string]interface{}{
				"qf":         "text",
				"q":          params.Keyword,
				"fore":       "{!type=$defType qf=$qf v=$q}",
				"back":       "*:*",
				"defType":    "edismax",
				"rows":       0,
				"echoParams": "none",
				"omitHeader": "true",
			},
			"facet": map[string]interface{}{
				"body": map[string]interface{}{
					"type":  "terms",
					"field": "text",
					"sort": map[string]interface{}{
						"relatedness": "desc",
					},
					"mincount": 2,
					"limit":    8,
					"facet": map[string]interface{}{
						"relatedness": map[string]interface{}{
							"type": "func",
							"func": "relatedness($fore,$back)",
						},
					},
				},
			},
		}

		payload, err := json.Marshal(reqBody)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to marshal request"})
		}
		fmt.Println(string(payload))
		// Build Solr query URL.
		url := fmt.Sprintf("%s/%s/query", solrURL, params.Collection)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to send post request"})
		}
		defer resp.Body.Close()

		var solrResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&solrResp); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to decode response"})
		}
		b, _ := json.Marshal(solrResp)
		fmt.Println(string(b))
		// Parse the response buckets.
		facets, ok := solrResp["facets"].(map[string]interface{})
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid facets in response"})
		}
		body, ok := facets["body"].(map[string]interface{})
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid body in facets"})
		}
		buckets, ok := body["buckets"].([]interface{})
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "invalid buckets in response"})
		}

		result := make([]RelatedTerm, 0)
		for _, bucket := range buckets {
			bkt, ok := bucket.(map[string]interface{})
			if !ok {
				continue
			}
			var relatedVal, val string
			if rel, ok := bkt["relatedness"].(map[string]interface{}); ok {
				relatedVal = fmt.Sprintf("%v", rel["relatedness"])
			}
			val = fmt.Sprintf("%v", bkt["val"])
			// result.WriteString(fmt.Sprintf("%s\t%s\n", relatedVal, val))
			result = append(result, RelatedTerm{
				Term:        val,
				Relatedness: relatedVal,
			})
		}

		return c.JSON(http.StatusOK, result)
	}
}

// Node defines the parameters for a facet node.
type Node struct {
	Name            string   // optional; if empty a default name is assigned
	Values          []string // if non-empty, a query facet is used; otherwise a terms facet is used
	Field           string   // the field to query or faceting field
	MinOccurrence   *int     // optional mincount (if provided)
	Limit           *int     // optional limit on facet results; if nil a default is used
	MinPopularity   *int     // optional min_popularity to be applied on nested facet
	DefaultOperator string   // defaults to "AND" if empty
}

// generateRequestRoot creates the basic request structure.
func generateRequestRoot() map[string]interface{} {
	return map[string]interface{}{
		"limit": 0,
		"params": map[string]interface{}{
			"q":       "*:*",
			"fore":    "{!${defType} v=$q}",
			"back":    "*:*",
			"defType": "edismax",
		},
		"facet": map[string]interface{}{},
	}
}

// getDefaultOperator returns the operator or "AND" if empty.
func getDefaultOperator(op string) string {
	if op == "" {
		return "AND"
	}
	return op
}

// deepCopyMap returns a deep copy of a map.
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case map[string]interface{}:
			newMap[k] = deepCopyMap(val)
		default:
			newMap[k] = v
		}
	}
	return newMap
}

// generateFacets returns a slice of facet definitions based on the provided node parameters.
func generateFacets(name string, values []string, field string, minOccurrence *int, limit *int, minPopularity *int, defaultOperator string) []map[string]interface{} {
	// Choose facet type based on values.
	facetType := "terms"
	if len(values) > 0 {
		facetType = "query"
	}

	// Set limit default to 10 if not provided.
	facLimit := 10
	if limit != nil {
		facLimit = *limit
	}

	baseFacet := map[string]interface{}{
		"type":  facetType,
		"limit": facLimit,
		"sort": map[string]interface{}{
			"relatedness": "desc",
		},
		"facet": map[string]interface{}{
			"relatedness": map[string]interface{}{
				"type": "func",
				"func": "relatedness($fore,$back)",
			},
		},
	}
	if minOccurrence != nil {
		baseFacet["mincount"] = *minOccurrence
	}
	if minPopularity != nil {
		// Add to nested facet.
		if facetMap, ok := baseFacet["facet"].(map[string]interface{}); ok {
			if rel, ok := facetMap["relatedness"].(map[string]interface{}); ok {
				rel["min_popularity"] = *minPopularity
			}
		}
	}
	if field != "" {
		baseFacet["field"] = field
	}

	var facets []map[string]interface{}
	if len(values) > 0 {
		// When values are present, remove mincount (if any) as per the Python code.
		if minOccurrence != nil {
			delete(baseFacet, "mincount")
		}
		if limit == nil {
			delete(baseFacet, "limit")
		}
		// For each value, create a facet copy with the appropriate query.
		for i := range values {
			facetCopy := deepCopyMap(baseFacet)
			queryStr := fmt.Sprintf("{!edismax q.op=%s qf=%s v=$%s_%d_query}", getDefaultOperator(defaultOperator), field, name, i)
			facetCopy["query"] = queryStr
			facets = append(facets, facetCopy)
		}
	} else {
		facets = append(facets, baseFacet)
	}
	return facets
}

// defaultNodeName generates a default name based on indices.
func defaultNodeName(i, j int) string {
	if j == 0 {
		return fmt.Sprintf("f%d", i)
	}
	return fmt.Sprintf("f%d_%d", i, j)
}

// transformRequest generates a faceted Solr SKG request from one or more multi-nodes.
// Each multi-node can be either a single Node or a slice of Node.
// Subsequent nodes are nested as facets of their parent nodes.
func transformRequest(multiNodes ...interface{}) map[string]interface{} {
	request := generateRequestRoot()
	params := request["params"].(map[string]interface{})
	// Start with the root as the only parent node.
	parentNodes := []map[string]interface{}{request}

	for i, mNode := range multiNodes {
		var nodes []Node
		// If the multi-node is a single Node, wrap it in a slice.
		switch v := mNode.(type) {
		case Node:
			nodes = []Node{v}
		case []Node:
			nodes = v
		default:
			// Skip if the type is unrecognized.
			continue
		}
		var currentFacets []map[string]interface{}
		for j, node := range nodes {
			if node.Name == "" {
				node.Name = defaultNodeName(i, j)
			}
			facets := generateFacets(node.Name, node.Values, node.Field, node.MinOccurrence, node.Limit, node.MinPopularity, node.DefaultOperator)
			currentFacets = append(currentFacets, facets...)
			// Attach the generated facets to each parent node.
			for _, parentNode := range parentNodes {
				facetField, ok := parentNode["facet"].(map[string]interface{})
				if !ok {
					facetField = map[string]interface{}{}
					parentNode["facet"] = facetField
				}
				for k, facet := range facets {
					key := fmt.Sprintf("%s_%d", node.Name, k)
					facetField[key] = facet
				}
			}
			// If the node has a values array, add query parameters to the root.
			if len(node.Values) > 0 {
				for k, value := range node.Values {
					key := fmt.Sprintf("%s_%d_query", node.Name, k)
					params[key] = value
				}
			}
		}
		// Update parentNodes to be the facets from this multi-node.
		parentNodes = currentFacets
	}
	return request
}

// To help with debugging, you can marshal the request structure to JSON.
func requestToJSON(request map[string]interface{}) (string, error) {
	b, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

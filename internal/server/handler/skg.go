package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
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

		reqBody := transformRequest(Node{
			Field: "text",
			Values: []string{
				params.Keyword,
			},
		}, Node{
			Field:         "text",
			MinOccurrence: lo.ToPtr(2),
			Limit:         lo.ToPtr(8),
		})

		payload, err := json.Marshal(reqBody)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to marshal request"})
		}
		fmt.Println(string(payload))
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
		result := transformResponseFacet(solrResp["facets"].(map[string]interface{}), reqBody["params"].(map[string]interface{}))
		fmt.Println(result)

		r := result[0]
		r2 := r["values"].([]interface{})
		r3 := r2[0].(map[string]interface{})[params.Keyword].(map[string]interface{})
		r4 := r3["traversals"].([]interface{})
		fmt.Println(r4[0].(map[string]interface{})["values"])

		// fmt.Println(result[0]["values"][0][params.Keyword]["traversals"]["values"])
		// var relatedTerms []RelatedTerm
		// for _, r := range result {
		// 	for _, v := range r["values"].(map[string]interface{}) {
		// 		relatedTerms = append(relatedTerms, RelatedTerm{
		// 			Term:        v.(map[string]interface{})["relatedness"].(string),
		// 			Relatedness: r["relatedness"].(string),
		// 		})
		// 	}
		// }

		return c.JSON(http.StatusOK, result)
	}
}

func extract(input []map[string]interface{}, t []RelatedTerm) {

	for _, m := range input {
		for k, v := range m {
			if k == "traversals" {
				t = append(t, RelatedTerm{
					Term:        "",
					Relatedness: v.(string),
				})
			}
		}
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
					parentNode["facet"].(map[string]interface{})[key] = facet
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

// transformNode converts a single node from the response.
// It retrieves the relatedness value (if count > 0) and processes nested traversals.
func transformNode(node map[string]interface{}, responseParams map[string]interface{}) map[string]interface{} {
	var relatedness float64 = 0.0

	// Only set relatedness if "count" > 0.
	if countVal, ok := node["count"]; ok {
		switch cnt := countVal.(type) {
		case float64:
			if cnt > 0 {
				if relMap, ok := node["relatedness"].(map[string]interface{}); ok {
					if relVal, ok := relMap["relatedness"]; ok {
						if r, ok := relVal.(float64); ok {
							relatedness = r
						}
					}
				}
			}
		}
	}

	valueNode := map[string]interface{}{
		"relatedness": relatedness,
	}

	subTraversals := transformResponseFacet(node, responseParams)
	if len(subTraversals) > 0 {
		valueNode["traversals"] = subTraversals
	}

	return valueNode
}

// transformResponseFacet traverses a response node and groups all sub-facets
// into traversal maps while skipping certain ignored keys.
func transformResponseFacet(node map[string]interface{}, responseParams map[string]interface{}) []map[string]interface{} {
	ignoredKeys := map[string]bool{
		"count":       true,
		"relatedness": true,
		"val":         true,
	}
	traversals := make(map[string]map[string]interface{})

	for fullName, data := range node {
		if ignoredKeys[fullName] {
			continue
		}

		// Remove the trailing suffix: "_" + the last segment.
		name := removeSuffix(fullName)

		// Initialize traversal if not already present.
		if _, exists := traversals[name]; !exists {
			traversals[name] = map[string]interface{}{
				"name":   name,
				"values": map[string]interface{}{},
			}
		}

		// Ensure data is a map.
		if dataMap, ok := data.(map[string]interface{}); ok {
			// If there is a "buckets" key then process each bucket.
			if buckets, ok := dataMap["buckets"]; ok {
				if bucketList, ok := buckets.([]interface{}); ok {
					valuesNode := make(map[string]interface{})
					for _, b := range bucketList {
						if bucket, ok := b.(map[string]interface{}); ok {
							// Use the "val" field as key.
							keyStr := fmt.Sprintf("%v", bucket["val"])
							valuesNode[keyStr] = transformNode(bucket, responseParams)
						}
					}
					traversals[name]["values"] = valuesNode
				}
			} else {
				// Otherwise, use responseParams to get the query value.
				queryKey := fmt.Sprintf("%s_query", fullName)
				valueName := ""
				if v, ok := responseParams[queryKey]; ok {
					valueName = fmt.Sprintf("%v", v)
				}
				// Ensure "values" is a map.
				valuesMap, ok := traversals[name]["values"].(map[string]interface{})
				if !ok {
					valuesMap = make(map[string]interface{})
				}
				valuesMap[valueName] = transformNode(dataMap, responseParams)
				traversals[name]["values"] = valuesMap
			}
		}
	}

	// Sort each traversal's values by relatedness descending.
	for key, traversal := range traversals {
		if values, ok := traversal["values"].(map[string]interface{}); ok {
			sortedValues := sortByRelatednessDesc(values)
			traversals[key]["values"] = sortedValues
		}
	}

	// Convert map to slice.
	var result []map[string]interface{}
	for _, v := range traversals {
		result = append(result, v)
	}

	return result
}

// removeSuffix removes the trailing "_" plus the last segment from the string.
// For example "foo_1" becomes "foo". If no underscore is found, the input is returned.
func removeSuffix(s string) string {
	if idx := strings.LastIndex(s, "_"); idx != -1 {
		return s[:idx]
	}
	return s
}

// sortByRelatednessDesc sorts the provided map values by their "relatedness" field in descending order
// and returns a slice of the sorted values.
func sortByRelatednessDesc(m map[string]interface{}) []interface{} {
	type kv struct {
		key         string
		value       interface{}
		relatedness float64
	}

	var kvList []kv
	for k, v := range m {
		var r float64
		if valMap, ok := v.(map[string]interface{}); ok {
			if rel, ok := valMap["relatedness"]; ok {
				if rf, ok := rel.(float64); ok {
					r = rf
				}
			}
		}
		kvList = append(kvList, kv{key: k, value: v, relatedness: r})
	}

	sort.Slice(kvList, func(i, j int) bool {
		return kvList[i].relatedness > kvList[j].relatedness
	})

	var sorted []interface{}

	for _, kv := range kvList {
		sorted = append(sorted, kv)
	}
	return sorted
}

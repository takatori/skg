package solr

import (
	"fmt"
	"strings"

	"github.com/takatori/skg/internal/skg"
)

type ResponseConverter struct {
	RequestParams map[string]interface{}
}

// transformResponseFacet processes a response node and creates traversal maps
// by grouping related facets while ignoring specific keys.
func (r *ResponseConverter) transformResponseFacet(node map[string]interface{}) map[string]skg.Traversal {
	ignoredKeys := map[string]bool{
		"count":       true,
		"relatedness": true,
		"val":         true,
	}
	traversals := make(map[string]skg.Traversal)

	for fullName, data := range node {
		if ignoredKeys[fullName] {
			continue
		}

		name := removeSuffix(fullName)

		// Skip non-map data
		dataMap, ok := data.(map[string]interface{})
		if !ok {
			continue
		}

		// Initialize traversal if needed
		traversal, exists := traversals[name]
		if !exists {
			traversal = skg.Traversal{
				Name: name,
			}
		}

		// Process buckets if they exist
		if buckets, hasBuckets := dataMap["buckets"]; hasBuckets {
			traversal.Values = r.processBuckets(buckets)
		} else {
			// Process single node
			n := r.transformNode(dataMap)
			if valueName, ok := r.RequestParams[fmt.Sprintf("%s_query", fullName)]; ok {
				n.Key = valueName.(string)
			}

			traversal.Values = append(traversal.Values, n)
		}

		traversals[name] = traversal
	}

	return traversals
}

// processBuckets extracts and transforms bucket data into Node values
func (r *ResponseConverter) processBuckets(buckets interface{}) []skg.Node {
	bucketList, ok := buckets.([]interface{})
	if !ok {
		return nil
	}

	values := make([]skg.Node, 0, len(bucketList))
	for _, b := range bucketList {
		bucket, ok := b.(map[string]interface{})
		if !ok {
			continue
		}
		values = append(values, r.transformNode(bucket))
	}

	return values
}

// transformNode converts a response node into an skg.Node structure
// with key, relatedness value, and nested traversals.
func (r *ResponseConverter) transformNode(node map[string]interface{}) skg.Node {
	var keyStr string
	if val, ok := node["val"]; ok {
		keyStr = fmt.Sprintf("%v", val)
	} else {
		keyStr = ""
	}
	relatedness := extractRelatedness(node)

	valueNode := skg.Node{
		Key:         keyStr,
		Relatedness: relatedness,
	}

	// Process nested traversals
	subTraversals := r.transformResponseFacet(node)
	for _, subTraversal := range subTraversals {
		valueNode.Traversals = append(valueNode.Traversals, subTraversal)
	}

	return valueNode
}

// extractRelatedness retrieves the relatedness value from a node if available
func extractRelatedness(node map[string]interface{}) float64 {
	// Only extract relatedness if count > 0
	countVal, hasCount := node["count"]
	if !hasCount {
		return 0.0
	}

	count, ok := countVal.(float64)
	if !ok || count <= 0 {
		return 0.0
	}

	// Extract relatedness value
	relMap, ok := node["relatedness"].(map[string]interface{})
	if !ok {
		return 0.0
	}

	relVal, ok := relMap["relatedness"]
	if !ok {
		return 0.0
	}

	relatedness, ok := relVal.(float64)
	if !ok {
		return 0.0
	}

	return relatedness
}

// removeSuffix removes the trailing "_" plus the last segment from a string.
// For example "foo_1" becomes "foo". If no underscore is found, returns the original string.
func removeSuffix(s string) string {
	if idx := strings.LastIndex(s, "_"); idx != -1 {
		return s[:idx]
	}
	return s
}

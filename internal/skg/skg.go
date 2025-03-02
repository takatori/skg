package skg

import "context"

type Query struct {
	Name            string   // optional; if empty a default name is assigned
	Values          []string // if non-empty, a query facet is used; otherwise a terms facet is used
	Field           string   // the field to query or faceting field
	MinOccurrence   *int     // optional mincount (if provided)
	Limit           *int     // optional limit on facet results; if nil a default is used
	MinPopularity   *int     // optional min_popularity to be applied on nested facet
	DefaultOperator string   // defaults to "AND" if empty
}

type Traversal struct {
	Name   string
	Values []Node
}

type Node struct {
	Key         string
	Relatedness float64
	Traversals  []Traversal
}

type SemanticKnowledgeGraph interface {
	Traverse(context.Context, [][]Query, string) (map[string]Traversal, error)
}

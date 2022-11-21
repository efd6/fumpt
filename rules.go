package main

import (
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

// conventions contains the specific conventions for file classes in the
// package.
var conventions = map[string][]ast.Visitor{
	"_dev/build/build.yml": {
		canonicalQuotes{},
	},

	"changelog.yml": {
		canonicalQuotes{},
		canonicalOrder{},
	},
	"manifest.yml": {
		canonicalQuotes{},
		canonicalOrder{},
	},

	"data_stream/*/_dev/test/*/test-*-config.yml": {
		canonicalQuotes{},
		canonicalOrder{},
	},
	"data_stream/*/elasticsearch/ingest_pipeline/*.yml": {
		canonicalQuotes{},
		canonicalOrder{},
	},
	"data_stream/*/fields/*.yml": {
		canonicalQuotes{},
		canonicalOrder{},
		sortLists{
			canSort: isECSgroup,
			less:    lessByName,
		},
	},
	"data_stream/*/manifest.yml": {
		canonicalQuotes{},
		canonicalOrder{},
	},
}

// isECSgroup returns whether the node n is in a 'type: group' field.
func isECSgroup(root ast.Node, n *ast.SequenceNode) bool {
	owner := up(2, root, n)
	if owner == nil {
		// We can sort root lists.
		return up(1, root, n) == root
	}
	m, ok := owner.(*ast.MappingNode)
	if ok {
		for _, v := range m.Values {
			if v.Key.String() != "type" {
				continue
			}
			return v.Value.String() == "group"
		}
	}
	return false
}

// up returns the n-parent of child if it exists in the AST, or nil otherwise.
func up(n int, root, child ast.Node) ast.Node {
	for i := 0; i < n; i++ {
		prev := child
		child = ast.Parent(root, child)
		if child == prev {
			return nil
		}
	}
	return child
}

var namePath = mustPath(yaml.PathString("$.name"))

// lessByName returns whether the path of a is lexically less than
// the path of b, breaking ties by source order.
func lessByName(a, b ast.Node) bool {
	// Get name node.
	an, _ := namePath.FilterNode(a)
	bn, _ := namePath.FilterNode(b)
	switch {
	case an != nil && bn != nil:
		// Order lexically.
		return an.String() < bn.String()
	case an != nil:
		// Only the a node has a name so sort it first.
		return true
	case bn != nil:
		// Only the b node has a name so sort it first.
		return false
	default:
		// Fall back to source order.
		at := a.GetToken()
		bt := b.GetToken()
		switch {
		case at.Position.Line < bt.Position.Line:
			return true
		case at.Position.Line > bt.Position.Line:
			return false
		default:
			return at.Position.Column < bt.Position.Column
		}
	}
}

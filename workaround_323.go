package main

import (
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/token"
)

// TODO: When goccy/go-yaml#323 is fixed this file can be deleted and the
// build and tests repaired.

const (
	workAroundGoYAML323 = true
	skipReasonGoYAML323 = "see https://github.com/goccy/go-yaml/issues/323"
)

// mustQuoteValue returns whether the string node must be single quoted due to
// goccy/go-yaml#323
func mustQuoteValue(n *ast.StringNode, root ast.Node) (typ token.Type, yes bool) {
	parent, yes := up(1, root, n).(*ast.MappingValueNode)
	if !yes || parent.Value != n {
		return token.UnknownType, false
	}
	switch parent.Key.GetToken().Type {
	case token.SingleQuoteType, token.DoubleQuoteType:
		if n.Token.Type == token.SingleQuoteType || canSingleQuote(n) {
			typ = token.SingleQuoteType
		} else {
			typ = token.DoubleQuoteType
		}
		return typ, true
	}
	return token.UnknownType, false
}

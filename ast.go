package main

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/token"
)

func mustPath(p *yaml.Path, err error) *yaml.Path {
	if err != nil {
		panic(err)
	}
	return p
}

// canonicalQuotes is an ast.Visitor that canonicalises quote usage in the
// YAML source. If possible, the visitor will remove quotes, if this cannot
// be done, it will try to use single quotes in place of double quotes.
type canonicalQuotes struct {
	// root is the root node of the tree being walked.
	root ast.Node
}

func (v canonicalQuotes) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.StringNode:
		if typ, yes := mustQuoteValue(n, v.root); yes {
			n.Token.Type = typ
			break
		}

		if n.Token.Type != token.DoubleQuoteType && n.Token.Type != token.SingleQuoteType {
			break
		}
		switch {
		case canStripQuotes(n, v.root):
			n.Token.Type = token.StringType
		case canSingleQuote(n):
			n.Token.Type = token.SingleQuoteType
		}
	}
	return v
}

// canStripQuotes returns whether the token would be interpreted as a string
// with the same value if it had its quotes removed and it contains no special
// characters.
func canStripQuotes(n *ast.StringNode, root ast.Node) (ok bool) {
	if parent, ok := up(1, root, n).(*ast.MappingValueNode); ok && parent.Key == n {
		return !strings.ContainsAny(n.Token.Value, ` :{}[],&*#?|-<>=!%@\`+"\t\n")
	}
	tok := n.Token
	if strings.TrimSpace(tok.Value) != tok.Value || hasPrefixAny(tok.Value, `:{}[],&*#?|-<>=!%@\`) {
		return false
	}
	toks := lexer.Tokenize(tok.Value)
	if len(toks) == 0 {
		return false
	}
	return toks[0].Type == token.StringType && tok.Value == toks[0].Value
}

// hasPrefixAny returns whether s has a single rune prefix in chars. This
// function is required because goccy/go-yaml does not completely interpret
// field values.
func hasPrefixAny(s, chars string) bool {
	prefix, _ := utf8.DecodeRuneInString(s)
	if prefix == utf8.RuneError {
		return false
	}
	for _, r := range chars {
		if prefix == r {
			return true
		}
	}
	return false
}

// canSingleQuote returns whether the string node would be interpreted as a string
// with the same value if it had its double quotes replaced with single quotes.
func canSingleQuote(n *ast.StringNode) (ok bool) {
	if n.Token.Type == token.SingleQuoteType {
		return false
	}
	new := *n
	new.Token = token.SingleQuote(n.Token.Value, n.Token.Origin, n.Token.Position)
	return stripQuotes(n.String()) == stripQuotes(new.String())
}

func stripQuotes(s string) string {
	switch {
	case len(s) < 2:
		return s
	case strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`),
		strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`):
		return s[1 : len(s)-1]
	default:
		return s
	}
}

// canonicalOrder is an ast.Visitor that canonicalises the ordering of YAML
// map fields. The ordering used is specified in the map by look-up of the
// field YAML path to an ordering priority. The syntax uses an extension of
// YAML path to allow priorities to be assigned to unrooted instances in the
// YAML tree.
//
// Field sorting behaviour is defined by priority with lower value priority
// fields being sorted before higher values. Negative priority values are
// sorted last. Field paths without an assigned priority sort between
// non-negative priorities and negative priorities. Ordering between fields
// with the same priority value is resolved by lexical ordering.
//
// Example syntax:
//
// Sort used to order a changelog file keeping the version first, followed
// by the changes map, with the ordering of each change starting with the
// description, then type and link. All path are rooted ('$') and changes
// are applied to all elements of each sequence ('[*]').
//
//  canonicalOrder{
//  	"$[*].version":                0,
//  	"$[*].changes":                1,
//  	"$[*].changes[*].description": 0,
//  	"$[*].changes[*].type":        1,
//  	"$[*].changes[*].link":        2,
//  }
//
// Sort used to order a package manifest file. This configuration keeps the
// owner field last and places all 'name' fields first, no matter the location
// of the 'name' field by using the unrooted syntax ('*').
//
//  canonicalOrder{
//  	"*.name":        0,
//  	"*.title":       1,
//  	"$.version":     2,
//  	"*.description": 3,
//  	"$.owner":       -1,
//  }
//
type canonicalOrder map[string]int

func (v canonicalOrder) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.MappingNode:
		sort.Slice(n.Values, func(i, j int) bool {
			pi := replaceIndices(n.Values[i].Key.GetPath())
			pj := replaceIndices(n.Values[j].Key.GetPath())
			oi, oki := v.ordering(pi)
			oj, okj := v.ordering(pj)
			switch {
			case oki && okj:
				// We have a user defined ordering for the two.
				if (oi < 0) == (oj < 0) {
					// ith element sorts before the jth
					// element by ordered comparison
					// of their ordering value if they
					// match signs, breaking ties lexically
					// by path.
					switch {
					case oi < oj:
						return true
					case oi > oj:
						return false
					default:
						return pi < pj
					}
				} else {
					// Otherwise the ith element sorts first
					// if it is non-negative.
					return oi >= 0
				}
			case oki:
				// Only the ith key has an ordering, so it sorts
				// first unless it has a negative ordering.
				return oi >= 0
			case okj:
				// Only the jth key has an ordering, so it sorts
				// first unless it has a negative ordering.
				return oj < 0
			default:
				// Fall back to lexical ordering.
				return pi < pj
			}
		})
	}
	return v
}

var indices = regexp.MustCompile(`\[[0-9]+\]`)

func replaceIndices(s string) string {
	return indices.ReplaceAllString(s, "[*]")
}

func (v canonicalOrder) ordering(s string) (order int, ok bool) {
	order, ok = v[s]
	if ok {
		return order, ok
	}
	for {
		idx := strings.Index(s, ".")
		if idx == -1 {
			break
		}
		order, ok = v["*"+s[idx:]]
		if ok {
			return order, ok
		}
		s = s[idx+1:]
	}
	return 0, false
}

// sortLists is an ast.Visitor that canonicalises list ordering in the YAML
// source. Because ordering of lists may be semantically laden sorting is
// conditional.
type sortLists struct {
	// root is the root node of the tree being walked.
	root ast.Node

	// canSort returns whether the node may be sorted
	// according to the semantics of the YAML source.
	canSort func(root ast.Node, node *ast.SequenceNode) bool

	// less implements the list ordering to use.
	less func(a, b ast.Node) bool
}

func (v sortLists) Visit(n ast.Node) ast.Visitor {
	if v.less == nil {
		return nil
	}
	switch n := n.(type) {
	case *ast.SequenceNode:
		if v.canSort == nil || v.canSort(v.root, n) {
			sort.Slice(n.Values, func(i, j int) bool {
				return v.less(n.Values[i], n.Values[j])
			})
		}
	}
	return v
}

// indentVisitor is a work-around for a failure in goccy/go-yaml to
// correctly set indent of in-line JSON. See https://github.com/goccy/go-yaml/issues/324.
type indentVisitor struct{}

func (v indentVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.MappingValueNode:
		indent := n.GetToken().Position.IndentLevel
		if n.Key.GetToken().Position.IndentLevel <= indent {
			if n, ok := n.Key.(*ast.StringNode); ok {
				n.Token.Position.IndentLevel = indent + 1
			}
		}
	}
	return v
}

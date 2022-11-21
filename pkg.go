package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"golang.org/x/tools/txtar"
)

// walk does a file-system walk of the package rooted at root
// applying the rewrite rules corresponding to files relative
// to the root. If w is not nil, a txtar of the result is written
// to it.
func walk(root string, w io.Writer, rules map[string][]ast.Visitor) error {
	pkg := filepath.Base(root)
	var ar txtar.Archive
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yml" {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		visitors, ok := rules[classFor(rel)]
		if !ok {
			return nil
		}
		data, err := applyChanges(path, visitors)
		if err != nil {
			return err
		}

		if w == nil {
			err := os.WriteFile(path, []byte(data), 0o644)
			if err != nil {
				return err
			}
		} else {
			ar.Files = append(ar.Files, txtar.File{
				Name: filepath.Join(pkg, rel),
				Data: []byte(data),
			})
		}

		return nil
	})
	if err != nil {
		return err
	}

	if w != nil {
		_, err = w.Write(txtar.Format(&ar))
	}
	return err
}

var (
	starDataStream = regexp.MustCompile(`^data_stream/[^/]+/`)
	starTests      = regexp.MustCompile(`/(pipeline|system)/test-[^/]+-config.yml$`)
	starFields     = regexp.MustCompile(`/fields/[^/]+\.yml$`)
	starPipelines  = regexp.MustCompile(`/ingest_pipeline/[^/]+.yml$`)
)

// classFor returns the class of file corresponding to relative path of the
// file in rel.
func classFor(rel string) string {
	class := rel
	if strings.HasPrefix(class, "data_stream") {
		class = starDataStream.ReplaceAllString(class, "data_stream/*/")
		class = starTests.ReplaceAllString(class, "/*/test-*-config.yml")
		class = starFields.ReplaceAllString(class, "/fields/*.yml")
		class = starPipelines.ReplaceAllString(class, "/ingest_pipeline/*.yml")
	}
	return class
}

// applyChanges applies the changes specified by the visitors
// to the file at path, returning the result of the re-write.
func applyChanges(path string, visitors []ast.Visitor) (string, error) {
	file, err := parser.ParseFile(path, parser.ParseComments)
	if err != nil {
		return "", fmt.Errorf("failed to parse document: %w", err)
	}
	for _, doc := range file.Docs {
		ast.Walk(indentVisitor{}, doc)
		for _, v := range visitors {
			// Tell visitors that need to traverse
			// up about the tree root.
			switch u := v.(type) {
			case canonicalQuotes:
				u.root = doc
				v = u
			case sortLists:
				u.root = doc
				v = u
			}
			ast.Walk(v, doc)
		}
	}
	return file.String(), nil
}

// root finds the root of the package containing the directory provided.
func root(dir string) (string, error) {
	d, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		ok, err := isRootDir(d)
		if err != nil {
			return "", err
		}
		if ok {
			return d, nil
		}
		if d == "/" {
			return "", os.ErrNotExist
		}
		d = filepath.Dir(d)
	}
}

var typePath = mustPath(yaml.PathString("$.type"))

// isRootDir returns whether dir is a fleet package root directory.
func isRootDir(dir string) (ok bool, err error) {
	p := filepath.Join(dir, "manifest.yml")

	_, err = os.Stat(p)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	case err != nil:
		return false, err
	}

	f, err := parser.ParseFile(p, 0)
	if err != nil {
		return false, err
	}
	n, err := typePath.FilterFile(f)
	switch {
	case errors.Is(err, yaml.ErrNotFoundNode):
		fmt.Println(err)
		return false, nil
	case err != nil:
		return false, err
	}

	return n.String() == "integration", nil
}

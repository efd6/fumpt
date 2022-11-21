package main

import (
	"bytes"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "update test expectations")

func TestRoot(t *testing.T) {
	pkgPath := filepath.FromSlash("testdata/pkg")

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}

	err = filepath.WalkDir(pkgPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		got, err := root(path)
		if d.IsDir() {
			if err != nil {
				t.Errorf("unexpected error for %s: %v", path, err)
			}
			got, err := filepath.Rel(wd, got)
			if err != nil {
				t.Fatalf("failed to get relative path: %v", err)
			}
			if got != pkgPath {
				t.Errorf("unexpected root path for %s: got:%v want:%s", path, got, pkgPath)
			}
		} else {
			if err == nil {
				t.Errorf("expected error for %s: %v", path, err)
			}
			if got != "" {
				t.Errorf("unexpected root path for non-directory %s: got:%v", path, got)
			}
		}
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error during walk: %v", err)
	}
}

func TestWalk(t *testing.T) {
	var buf bytes.Buffer
	err := walk("testdata/pkg", &buf, conventions)
	if err != nil {
		t.Errorf("unexpected error during walk: %v", err)
	}
	if *update {
		err = os.WriteFile("pkg_want.txtar", buf.Bytes(), 0o644)
		if err != nil {
			t.Errorf("unexpected error writing testdata: %v", err)
		}
		return
	}
	want, err := os.ReadFile("pkg_want.txtar")
	if err != nil {
		t.Errorf("unexpected error reading testdata: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("unexpected result:\n--- got\n+++ want\n%s", cmp.Diff(buf.Bytes(), want))
	}
}

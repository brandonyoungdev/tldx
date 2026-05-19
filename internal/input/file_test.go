package input_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brandonyoungdev/tldx/internal/input"
)

func TestReadKeywordsFromFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "keywords.txt")

	content := "  apple  \n\nbanana\n  cherry \n\n"
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	keywords, err := input.ReadKeywordsFromFile(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"apple", "banana", "cherry"}

	if len(keywords) != len(expected) {
		t.Fatalf("expected %d keywords, got %d", len(expected), len(keywords))
	}

	for i, k := range keywords {
		if k != expected[i] {
			t.Errorf("expected keyword %q at index %d, got %q", expected[i], i, k)
		}
	}

	_, err = input.ReadKeywordsFromFile("nonexistent_file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestReadKeywordsFromStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	w.WriteString("foo\nbar\nbaz\n")
	w.Close()

	keywords, err := input.ReadKeywordsFromFile("-")
	if err != nil {
		t.Fatalf("unexpected error reading from stdin: %v", err)
	}

	expected := []string{"foo", "bar", "baz"}
	if len(keywords) != len(expected) {
		t.Fatalf("expected %d keywords, got %d: %v", len(expected), len(keywords), keywords)
	}
	for i, k := range keywords {
		if k != expected[i] {
			t.Errorf("expected %q at index %d, got %q", expected[i], i, k)
		}
	}
}

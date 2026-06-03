package cmd

import (
	"testing"
)

func TestWrapPresetText_ShortText_ReturnsSingle(t *testing.T) {
	got := wrapPresetText("com io dev", 100)
	if len(got) != 1 {
		t.Fatalf("expected 1 line for short text, got %d: %v", len(got), got)
	}
	if got[0] != "com io dev" {
		t.Errorf("expected %q, got %q", "com io dev", got[0])
	}
}

func TestWrapPresetText_LongText_WrapsAtWordBoundary(t *testing.T) {
	got := wrapPresetText("alpha beta gamma delta", 12)
	if len(got) < 2 {
		t.Fatalf("expected at least 2 lines for long text, got %d: %v", len(got), got)
	}
}

func TestWrapPresetText_LongSingleWord_NoSplit(t *testing.T) {
	got := wrapPresetText("superlongwordthatexceedsmaxwidth", 10)
	if len(got) != 1 {
		t.Fatalf("expected 1 line for single oversized word, got %d: %v", len(got), got)
	}
	if got[0] != "superlongwordthatexceedsmaxwidth" {
		t.Errorf("expected original word, got %q", got[0])
	}
}

func TestWrapPresetText_MultipleOversizedWords(t *testing.T) {
	got := wrapPresetText("longerword1 longerword2", 5)
	if len(got) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(got), got)
	}
}

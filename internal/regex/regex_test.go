package regex

import (
	"testing"
)

func TestExpandPattern(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		expectedCount int
		checkFirst    string
		checkLast     string
		expectError   bool
	}{
		{
			name:          "Single character with repetition",
			pattern:       "[a]{2}",
			expectedCount: 1,
			checkFirst:    "aa",
			checkLast:     "aa",
			expectError:   false,
		},
		{
			name:          "Two characters with repetition",
			pattern:       "[ab]{2}",
			expectedCount: 4, // aa, ab, ba, bb
			checkFirst:    "aa",
			checkLast:     "bb",
			expectError:   false,
		},
		{
			name:          "Three letter combinations",
			pattern:       "[a-c]{3}",
			expectedCount: 27, // 3^3
			checkFirst:    "aaa",
			checkLast:     "ccc",
			expectError:   false,
		},
		{
			name:          "Range with literal",
			pattern:       "x[a-b]{2}",
			expectedCount: 4, // xaa, xab, xba, xbb
			checkFirst:    "xaa",
			checkLast:     "xbb",
			expectError:   false,
		},
		{
			name:          "Multiple segments",
			pattern:       "[ab]{1}[cd]{1}",
			expectedCount: 4, // ac, ad, bc, bd
			expectError:   false,
		},
		{
			name:        "Unclosed character class",
			pattern:     "[a-z",
			expectError: true,
		},
		{
			name:        "Unclosed repetition",
			pattern:     "[a-z]{3",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExpandPattern(tt.pattern)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.checkFirst != "" && len(results) > 0 && results[0] != tt.checkFirst {
				t.Errorf("Expected first result to be %s, got %s", tt.checkFirst, results[0])
			}

			if tt.checkLast != "" && len(results) > 0 && results[len(results)-1] != tt.checkLast {
				t.Errorf("Expected last result to be %s, got %s", tt.checkLast, results[len(results)-1])
			}
		})
	}
}

func TestParseCharClass(t *testing.T) {
	tests := []struct {
		name          string
		class         string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Lowercase range",
			class:         "a-z",
			expectedCount: 26,
			expectError:   false,
		},
		{
			name:          "Numeric range",
			class:         "0-9",
			expectedCount: 10,
			expectError:   false,
		},
		{
			name:          "Mixed ranges",
			class:         "a-z0-9",
			expectedCount: 36,
			expectError:   false,
		},
		{
			name:          "Single characters",
			class:         "abc",
			expectedCount: 3,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chars, err := parseCharClass(tt.class)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(chars) != tt.expectedCount {
				t.Errorf("Expected %d characters, got %d", tt.expectedCount, len(chars))
			}
		})
	}
}

func TestIsPatternSafe(t *testing.T) {
	tests := []struct {
		name            string
		pattern         string
		maxCombinations int
		expectedSafe    bool
		expectedCount   int
	}{
		{
			name:            "Safe pattern - 2 letters",
			pattern:         "[a-z]{2}",
			maxCombinations: 1000,
			expectedSafe:    true,
			expectedCount:   676,
		},
		{
			name:            "Safe pattern - 3 letters",
			pattern:         "[a-z]{3}",
			maxCombinations: 20000,
			expectedSafe:    true,
			expectedCount:   17576,
		},
		{
			name:            "Unsafe pattern - 4 letters",
			pattern:         "[a-z]{4}",
			maxCombinations: 100000,
			expectedSafe:    false,
		},
		{
			name:            "Safe pattern with literal",
			pattern:         "x[a-z]{2}",
			maxCombinations: 1000,
			expectedSafe:    true,
			expectedCount:   676,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			safe, count, err := IsPatternSafe(tt.pattern, tt.maxCombinations)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if safe != tt.expectedSafe {
				t.Errorf("Expected safe=%v, got safe=%v (count=%d)", tt.expectedSafe, safe, count)
			}

			if tt.expectedSafe && count != tt.expectedCount {
				t.Errorf("Expected count=%d, got count=%d", tt.expectedCount, count)
			}
		})
	}
}

func TestGenerateSegmentCombos(t *testing.T) {
	tests := []struct {
		name          string
		chars         []rune
		count         int
		expectedCount int
		checkFirst    string
		checkLast     string
	}{
		{
			name:          "2 chars, count 2",
			chars:         []rune{'a', 'b'},
			count:         2,
			expectedCount: 4,
			checkFirst:    "aa",
			checkLast:     "bb",
		},
		{
			name:          "3 chars, count 1",
			chars:         []rune{'x', 'y', 'z'},
			count:         1,
			expectedCount: 3,
			checkFirst:    "x",
			checkLast:     "z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := generateSegmentCombos(tt.chars, tt.count)

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if len(results) > 0 && results[0] != tt.checkFirst {
				t.Errorf("Expected first result to be %s, got %s", tt.checkFirst, results[0])
			}

			if len(results) > 0 && results[len(results)-1] != tt.checkLast {
				t.Errorf("Expected last result to be %s, got %s", tt.checkLast, results[len(results)-1])
			}
		})
	}
}

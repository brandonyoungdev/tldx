package regex

import (
	"fmt"
	"regexp"
)

// ExpandPattern expands a regex pattern into all possible domain name combinations
// Supports: [a-z] character ranges, {n} repetition counts, and literal characters
// Example: "[a-z]{3}" generates all 3-letter combinations: aaa, aab, aac, ..., zzz
func ExpandPattern(pattern string) ([]string, error) {
	segments, err := parsePattern(pattern)
	if err != nil {
		return nil, err
	}

	results := generateCombinations(segments)
	return results, nil
}

type Segment struct {
	chars []rune
	count int
}

func parsePattern(pattern string) ([]Segment, error) {
	var segments []Segment
	i := 0
	runes := []rune(pattern)

	for i < len(runes) {
		switch runes[i] {
		case '[':
			// Parse character class
			j := i + 1
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j >= len(runes) {
				return nil, fmt.Errorf("unclosed character class")
			}

			chars, err := parseCharClass(string(runes[i+1 : j]))
			if err != nil {
				return nil, err
			}

			// Check for repetition
			count := 1
			if j+1 < len(runes) && runes[j+1] == '{' {
				k := j + 2
				for k < len(runes) && runes[k] != '}' {
					k++
				}
				if k >= len(runes) {
					return nil, fmt.Errorf("unclosed repetition")
				}
				var num int
				_, err := fmt.Sscanf(string(runes[j+2:k]), "%d", &num)
				if err != nil {
					return nil, fmt.Errorf("invalid repetition count: %v", err)
				}
				count = num
				i = k + 1
			} else {
				i = j + 1
			}

			segments = append(segments, Segment{chars: chars, count: count})
		case '\\':
			// Escaped character
			if i+1 >= len(runes) {
				return nil, fmt.Errorf("incomplete escape sequence")
			}
			segments = append(segments, Segment{chars: []rune{runes[i+1]}, count: 1})
			i += 2
		case '{':
			return nil, fmt.Errorf("repetition without preceding element")
		default:
			// Literal character
			segments = append(segments, Segment{chars: []rune{runes[i]}, count: 1})
			i++
		}
	}

	return segments, nil
}

// parseCharClass parses a character class like "a-z" or "a-z0-9"
func parseCharClass(class string) ([]rune, error) {
	var chars []rune
	runes := []rune(class)
	i := 0

	for i < len(runes) {
		if i+2 < len(runes) && runes[i+1] == '-' {
			// Range
			start := runes[i]
			end := runes[i+2]
			if start > end {
				return nil, fmt.Errorf("invalid range: %c-%c", start, end)
			}
			for c := start; c <= end; c++ {
				chars = append(chars, c)
			}
			i += 3
		} else {
			// Single character
			chars = append(chars, runes[i])
			i++
		}
	}

	return chars, nil
}

func generateCombinations(segments []Segment) []string {
	if len(segments) == 0 {
		return []string{""}
	}

	// Start with empty string
	results := []string{""}

	// Process each segment
	for _, segment := range segments {
		var newResults []string
		// For each existing result
		for _, result := range results {
			// Generate all combinations for this segment's count
			combos := generateSegmentCombos(segment.chars, segment.count)
			for _, combo := range combos {
				newResults = append(newResults, result+combo)
			}
		}
		results = newResults
	}

	return results
}

func generateSegmentCombos(chars []rune, count int) []string {
	if count == 0 {
		return []string{""}
	}

	var results []string
	for _, c := range chars {
		subCombos := generateSegmentCombos(chars, count-1)
		for _, sub := range subCombos {
			results = append(results, string(c)+sub)
		}
	}

	return results
}

func IsPatternSafe(pattern string, maxCombinations int) (bool, int, error) {
	segments, err := parsePattern(pattern)
	if err != nil {
		return false, 0, err
	}

	total := 1
	for _, segment := range segments {
		chars := len(segment.chars)
		for range segment.count {
			total *= chars
			if total > maxCombinations {
				return false, total, nil
			}
		}
	}

	return true, total, nil
}

func ValidatePattern(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}

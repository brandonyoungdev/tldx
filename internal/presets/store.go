package presets

import (
	"fmt"
	"maps"
	"sort"
	"strings"
)

type PresetStore[T any] struct {
	Kind    string
	builtin map[string]T
	user    map[string]T
}

func NewTypedStore[T any](kind string, builtin map[string]T) *PresetStore[T] {
	return &PresetStore[T]{
		Kind:    kind,
		builtin: builtin,
		user:    make(map[string]T),
	}
}

func (ps *PresetStore[T]) Override(name string, value T) {
	ps.user[name] = value
}

func (ps *PresetStore[T]) Get(name string) (T, bool) {
	if val, ok := ps.user[name]; ok {
		return val, true
	}
	val, ok := ps.builtin[name]
	return val, ok
}

func (ps *PresetStore[T]) All() map[string]T {
	out := make(map[string]T)
	maps.Copy(out, ps.builtin)
	maps.Copy(out, ps.user)
	return out
}

func ShowAllPresets[T any](ps *PresetStore[T], stringify func(T) string) {
	all := ps.All()

	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)

	const maxWidth = 70
	const labelWidth = 22

	fmt.Printf("\n%s Presets:\n\n", strings.ToUpper(ps.Kind))
	fmt.Printf("%-*s  %s\n\n", labelWidth, "all", "(use all available TLDs)")

	for _, name := range names {
		displayName := strings.ToLower(name)
		tlds := stringify(all[name])

		// Wrap long TLD lists
		if len(tlds) > maxWidth-labelWidth-4 {
			lines := wrapText(tlds, maxWidth-labelWidth-4)
			fmt.Printf("%-*s  %s\n", labelWidth, displayName, lines[0])
			for i := 1; i < len(lines); i++ {
				fmt.Printf("%-*s  %s\n", labelWidth, "", lines[i])
			}
		} else {
			fmt.Printf("%-*s  %s\n", labelWidth, displayName, tlds)
		}
		fmt.Println()
	}
	fmt.Printf("%-*s  %s\n\n", labelWidth, "all", "(use all available TLDs)")
}

// wrapText wraps text at word boundaries to fit within maxWidth
func wrapText(text string, maxWidth int) []string {
	if len(text) <= maxWidth {
		return []string{text}
	}

	var lines []string
	words := strings.Split(text, " ")
	currentLine := ""

	for i, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		if len(testLine) > maxWidth {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Single word is longer than maxWidth
				lines = append(lines, word)
			}
		} else {
			currentLine = testLine
		}

		if i == len(words)-1 && currentLine != "" {
			lines = append(lines, currentLine)
		}
	}

	return lines
}

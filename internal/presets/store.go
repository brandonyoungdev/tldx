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

	fmt.Printf("\n== %s Presets ==\n\n", strings.ToTitle(ps.Kind))
	for _, name := range names {
		fmt.Printf("- %s: %s\n", name, stringify(all[name]))
	}
}

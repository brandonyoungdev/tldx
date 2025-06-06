package presets

import "slices"

var TLDs = NewTypedStore("tld", DefaultTLDPresets)

func GetAllTLDs() []string {
	allPresets := TLDs.All()
	var allTlds []string
	for _, tlds := range allPresets {
		allTlds = append(allTlds, tlds...)
	}

	slices.Sort(allTlds)
	allTlds = slices.Compact(allTlds)

	return allTlds
} 
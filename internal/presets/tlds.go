package presets

import "slices"

var TLDs = NewTypedStore("tld", DefaultTLDPresets)

func GetAllTLDs() []string {
	var all []string
	presets := TLDs.All()

	for _, tlds := range presets {
		all = append(all, tlds...)
	}

	slices.Sort(all)
	all = slices.Compact(all)

	return all
}

package strutil

import "strings"

func RemoveDuplicates(strs []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strs {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func AllToLowerCase(strs []string) []string {
	for i, str := range strs {
		strs[i] = strings.ToLower(str)
	}
	return strs
}

func FilterByMaxLength(strs []string, maxLength int) []string {
	var result []string
	for _, str := range strs {
		if len(str) <= maxLength {
			result = append(result, str)
		}
	}
	return result
}

package input

import (
	"bufio"
	"os"
	"strings"
)

func ReadKeywordsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keywords []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			keywords = append(keywords, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return keywords, nil
}

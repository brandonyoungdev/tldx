package input

import (
	"bufio"
	"io"
	"os"
	"strings"
)

func ReadKeywordsFromFile(filename string) ([]string, error) {
	var r io.Reader
	if filename == "-" {
		r = os.Stdin
	} else {
		file, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		r = file
	}

	var keywords []string
	scanner := bufio.NewScanner(r)
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

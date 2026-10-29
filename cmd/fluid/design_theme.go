package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/unaquatic/fluid/style"
)

// loadTheme returns the theme in the file at path, a style.Theme as JSON, or
// style.DarculaSolid when path is empty.
func loadTheme(path string) (*style.Theme, error) {
	if path == "" {
		return style.DarculaSolid(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var th style.Theme
	if err := json.Unmarshal(data, &th); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &th, nil
}

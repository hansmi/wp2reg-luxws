package main

import (
	"regexp"
)

// For some reason `0x` can be doubled in the output.
var hexRe = regexp.MustCompile(`^(0x)+`)

func normalizeId(text string) string {
	return hexRe.ReplaceAllString(text, "")
}

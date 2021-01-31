package main

import (
	"regexp"
	"strings"
)

var spaceRe = regexp.MustCompile(`\s+`)

func normalizeSpace(text string) string {
	return spaceRe.ReplaceAllString(strings.TrimSpace(text), " ")
}

package main

import (
	"slices"
	"strings"
)

var emptyJSONValues = []string{
	"null", `""`, "[]", "{}", "",
}

func IsEmptyJSON(s string) bool {
	s = strings.TrimSpace(s)
	return slices.Contains(emptyJSONValues, s)
}

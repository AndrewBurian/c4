package lexer

import "strings"

var knownKeywords = []string{
	"workspace",
	"extends",
	"model",
	"views",

	"person",
	"softwaresystem",
	"container",
	"component",
	"group",

	"perspectives",
	"tags",
	"description",
	"name",
	"properties",
	"technology",
	"url",
	"this",

	"style",
}

func isKeyword(s string) bool {
	s = strings.ToLower(s)
	for _, k := range knownKeywords {
		if s == k {
			return true
		}
	}
	return false
}

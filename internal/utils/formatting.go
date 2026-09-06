package utils

import (
	"html"
	"strings"
)

func EscapeHTML(s string) string {
	return html.EscapeString(s)
}

func CleanString(s string) string {
	return strings.TrimSpace(s)
}

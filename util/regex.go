package util

import "regexp"

// CodeBlockStrip removes fenced code blocks (backtick and tilde) from markdown.
var CodeBlockStrip = regexp.MustCompile("(?s)(?:```|~~~)[\\w]*\\n.*?(?:```|~~~)")

// InlineCodeStrip removes inline code spans from markdown.
var InlineCodeStrip = regexp.MustCompile("`[^`]+`")

// CodeBlockPattern extracts fenced code block bodies (capture group 1) from markdown.
var CodeBlockPattern = regexp.MustCompile("(?s)(?:```|~~~)[\\w]*\\n(.*?)(?:```|~~~)")

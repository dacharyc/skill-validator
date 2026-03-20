// Package content analyzes the textual content of SKILL.md files. It computes
// metrics such as word count, code block ratio, imperative sentence ratio,
// information density, and instruction specificity to assess content quality.
package content

import (
	"regexp"
	"strings"

	"github.com/agent-ecosystem/skill-validator/types"
	"github.com/agent-ecosystem/skill-validator/util"
)

// strongMarkerRes contains pre-compiled patterns for strong directive language
// markers (must, always, never, etc.) used to measure instruction specificity.
var strongMarkerRes = compilePatterns([]string{
	`\bmust\b`, `\balways\b`, `\bnever\b`, `\bshall\b`,
	`\brequired\b`, `\bdo not\b`, `\bdon't\b`, `\bensure\b`,
	`\bcritical\b`, `\bmandatory\b`,
})

// weakMarkerRes contains pre-compiled patterns for weak/advisory language
// markers (may, consider, could, etc.) used to measure instruction specificity.
var weakMarkerRes = compilePatterns([]string{
	`\bmay\b`, `\bconsider\b`, `\bcould\b`, `\bmight\b`,
	`\boptional\b`, `\bpossibly\b`, `\bsuggested\b`,
	`\bprefer\b`, `\btry to\b`, `\bif possible\b`,
})

func compilePatterns(patterns []string) []*regexp.Regexp {
	res := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		res[i] = regexp.MustCompile(p)
	}
	return res
}

// imperativeVerbs is the set of common imperative verbs used to detect
// instruction-style sentences in skill content.
var imperativeVerbs = map[string]bool{
	"use": true, "run": true, "create": true, "add": true, "set": true,
	"install": true, "configure": true, "write": true, "read": true,
	"check": true, "verify": true, "make": true, "build": true, "test": true,
	"ensure": true, "include": true, "remove": true, "delete": true,
	"update": true, "call": true, "import": true, "export": true,
	"define": true, "implement": true, "return": true, "pass": true,
	"handle": true, "parse": true, "generate": true, "format": true,
	"validate": true, "convert": true, "follow": true, "apply": true,
	"start": true, "stop": true, "avoid": true, "keep": true, "do": true,
	"execute": true, "open": true, "close": true, "save": true, "load": true,
	"send": true, "receive": true,
}

var (
	codeLangPattern  = regexp.MustCompile("(?:```|~~~)(\\w+)")
	sentenceSplitPat = regexp.MustCompile(`[.!?]\s+|[.!?]$|\n\n+`)
	leadingFormatPat = regexp.MustCompile(`^[#*\->\s]+`)
	sectionPattern   = regexp.MustCompile(`(?m)^#{2,}\s+`)
	listItemPattern  = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+|^\s*\d+\.\s+`)
)

// Analyze computes content metrics for SKILL.md content.
func Analyze(content string) *types.ContentReport {
	if strings.TrimSpace(content) == "" {
		return &types.ContentReport{}
	}

	words := strings.Fields(content)
	wordCount := len(words)

	// Code block analysis
	codeBlocks := util.CodeBlockPattern.FindAllStringSubmatch(content, -1)
	codeBlockCount := len(codeBlocks)
	codeBlockWords := 0
	for _, match := range codeBlocks {
		codeBlockWords += len(strings.Fields(match[1]))
	}
	codeBlockRatio := 0.0
	if wordCount > 0 {
		codeBlockRatio = float64(codeBlockWords) / float64(wordCount)
	}

	// Code languages
	langMatches := codeLangPattern.FindAllStringSubmatch(content, -1)
	codeLanguages := make([]string, 0, len(langMatches))
	for _, m := range langMatches {
		codeLanguages = append(codeLanguages, m[1])
	}

	// Sentence analysis
	sentences := countSentences(content)
	sentenceCount := len(sentences)
	imperativeCount := countImperativeSentences(sentences)
	imperativeRatio := 0.0
	if sentenceCount > 0 {
		imperativeRatio = float64(imperativeCount) / float64(sentenceCount)
	}

	// Information density: when code blocks are present, factor in the
	// code-to-prose ratio; otherwise score purely on imperative ratio so
	// prose-only skills aren't penalized for lacking code.
	informationDensity := imperativeRatio
	if codeBlockCount > 0 {
		informationDensity = (codeBlockRatio * 0.5) + (imperativeRatio * 0.5)
	}

	// Language marker analysis
	strongCount := countMarkerMatches(content, strongMarkerRes)
	weakCount := countMarkerMatches(content, weakMarkerRes)
	totalMarkers := strongCount + weakCount
	instructionSpecificity := 0.0
	if totalMarkers > 0 {
		instructionSpecificity = float64(strongCount) / float64(totalMarkers)
	}

	// Section count (H2+ headers)
	sectionCount := len(sectionPattern.FindAllString(content, -1))

	// List item count
	listItemCount := len(listItemPattern.FindAllString(content, -1))

	return &types.ContentReport{
		WordCount:              wordCount,
		CodeBlockCount:         codeBlockCount,
		CodeBlockRatio:         util.RoundTo(codeBlockRatio, 4),
		CodeLanguages:          codeLanguages,
		SentenceCount:          sentenceCount,
		ImperativeCount:        imperativeCount,
		ImperativeRatio:        util.RoundTo(imperativeRatio, 4),
		InformationDensity:     util.RoundTo(informationDensity, 4),
		StrongMarkers:          strongCount,
		WeakMarkers:            weakCount,
		InstructionSpecificity: util.RoundTo(instructionSpecificity, 4),
		SectionCount:           sectionCount,
		ListItemCount:          listItemCount,
	}
}

func countSentences(text string) []string {
	// Remove code blocks first
	text = util.CodeBlockStrip.ReplaceAllString(text, "")
	// Remove inline code
	text = util.InlineCodeStrip.ReplaceAllString(text, "")
	// Split on sentence boundaries
	parts := sentenceSplitPat.Split(text, -1)
	var sentences []string
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s != "" && len(s) > 5 {
			sentences = append(sentences, s)
		}
	}
	return sentences
}

func countImperativeSentences(sentences []string) int {
	count := 0
	for _, sentence := range sentences {
		// Get first word, stripping markdown formatting
		cleaned := leadingFormatPat.ReplaceAllString(sentence, "")
		words := strings.Fields(cleaned)
		if len(words) == 0 {
			continue
		}
		firstWord := strings.ToLower(words[0])
		if imperativeVerbs[firstWord] {
			count++
		}
	}
	return count
}

func countMarkerMatches(text string, patterns []*regexp.Regexp) int {
	total := 0
	textLower := strings.ToLower(text)
	for _, re := range patterns {
		total += len(re.FindAllString(textLower, -1))
	}
	return total
}

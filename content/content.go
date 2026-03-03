package content

import (
	"math"
	"regexp"
	"strings"
)

// Strong directive language markers
var strongMarkers = []string{
	`\bmust\b`, `\balways\b`, `\bnever\b`, `\bshall\b`,
	`\brequired\b`, `\bdo not\b`, `\bdon't\b`, `\bensure\b`,
	`\bcritical\b`, `\bmandatory\b`,
}

// Weak/advisory language markers
var weakMarkers = []string{
	`\bmay\b`, `\bconsider\b`, `\bcould\b`, `\bmight\b`,
	`\boptional\b`, `\bpossibly\b`, `\bsuggested\b`,
	`\bprefer\b`, `\btry to\b`, `\bif possible\b`,
}

// Common imperative verbs for instructions
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
	codeBlockPattern = regexp.MustCompile("(?s)```[\\w]*\\n(.*?)```")
	codeLangPattern  = regexp.MustCompile("```(\\w+)")
	codeBlockStrip   = regexp.MustCompile("(?s)```[\\w]*\\n.*?```")
	inlineCodeStrip  = regexp.MustCompile("`[^`]+`")
	sentenceSplitPat = regexp.MustCompile(`[.!?]\s+|[.!?]$|\n\n+`)
	leadingFormatPat = regexp.MustCompile(`^[#*\->\s]+`)
	sectionPattern   = regexp.MustCompile(`(?m)^#{2,}\s+`)
	listItemPattern  = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+|^\s*\d+\.\s+`)
)

// Report holds content analysis metrics for a skill.
type Report struct {
	WordCount              int      `json:"word_count"`
	CodeBlockCount         int      `json:"code_block_count"`
	CodeBlockRatio         float64  `json:"code_block_ratio"`
	CodeLanguages          []string `json:"code_languages"`
	SentenceCount          int      `json:"sentence_count"`
	ImperativeCount        int      `json:"imperative_count"`
	ImperativeRatio        float64  `json:"imperative_ratio"`
	InformationDensity     float64  `json:"information_density"`
	StrongMarkers          int      `json:"strong_markers"`
	WeakMarkers            int      `json:"weak_markers"`
	InstructionSpecificity float64  `json:"instruction_specificity"`
	SectionCount           int      `json:"section_count"`
	ListItemCount          int      `json:"list_item_count"`
}

// Analyze computes content metrics for SKILL.md content.
func Analyze(content string) *Report {
	if strings.TrimSpace(content) == "" {
		return &Report{}
	}

	words := strings.Fields(content)
	wordCount := len(words)

	// Code block analysis
	codeBlocks := codeBlockPattern.FindAllStringSubmatch(content, -1)
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
	strongCount := countMarkerMatches(content, strongMarkers)
	weakCount := countMarkerMatches(content, weakMarkers)
	totalMarkers := strongCount + weakCount
	instructionSpecificity := 0.0
	if totalMarkers > 0 {
		instructionSpecificity = float64(strongCount) / float64(totalMarkers)
	}

	// Section count (H2+ headers)
	sectionCount := len(sectionPattern.FindAllString(content, -1))

	// List item count
	listItemCount := len(listItemPattern.FindAllString(content, -1))

	return &Report{
		WordCount:              wordCount,
		CodeBlockCount:         codeBlockCount,
		CodeBlockRatio:         roundTo(codeBlockRatio, 4),
		CodeLanguages:          codeLanguages,
		SentenceCount:          sentenceCount,
		ImperativeCount:        imperativeCount,
		ImperativeRatio:        roundTo(imperativeRatio, 4),
		InformationDensity:     roundTo(informationDensity, 4),
		StrongMarkers:          strongCount,
		WeakMarkers:            weakCount,
		InstructionSpecificity: roundTo(instructionSpecificity, 4),
		SectionCount:           sectionCount,
		ListItemCount:          listItemCount,
	}
}

func countSentences(text string) []string {
	// Remove code blocks first
	text = codeBlockStrip.ReplaceAllString(text, "")
	// Remove inline code
	text = inlineCodeStrip.ReplaceAllString(text, "")
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

func countMarkerMatches(text string, patterns []string) int {
	total := 0
	textLower := strings.ToLower(text)
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		total += len(re.FindAllString(textLower, -1))
	}
	return total
}

func roundTo(val float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(val*pow) / pow
}

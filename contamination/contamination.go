// Package contamination detects cross-language and cross-technology
// contamination in skill content. It analyzes code block languages,
// technology references, and multi-interface tool usage to compute a
// contamination score indicating how likely a skill is to confuse an
// agent by mixing unrelated languages or domains.
package contamination

import (
	"math"
	"sort"
	"strings"

	"github.com/agent-ecosystem/skill-validator/types"
	"github.com/agent-ecosystem/skill-validator/util"
)

// multiInterfaceTools maps tool/platform names to the language identifiers
// commonly used with them. When a skill mentions one of these tools, multiple
// code languages are expected and should not be penalized as contamination.
var multiInterfaceTools = map[string][]string{
	"mongodb":       {"javascript", "python", "java", "csharp", "go", "ruby", "rust", "shell", "bash", "mongosh"},
	"aws":           {"python", "javascript", "typescript", "java", "go", "cli", "bash", "shell", "cloudformation", "terraform"},
	"docker":        {"yaml", "bash", "shell", "dockerfile", "python", "javascript"},
	"kubernetes":    {"yaml", "bash", "shell", "go", "python"},
	"redis":         {"python", "javascript", "java", "go", "ruby", "bash", "shell"},
	"postgresql":    {"sql", "python", "javascript", "java", "go", "ruby"},
	"mysql":         {"sql", "python", "javascript", "java", "go", "ruby"},
	"elasticsearch": {"json", "python", "javascript", "java", "curl", "bash"},
	"firebase":      {"javascript", "typescript", "python", "java", "swift", "kotlin", "dart"},
	"terraform":     {"hcl", "bash", "shell", "json", "yaml"},
	"graphql":       {"graphql", "javascript", "typescript", "python", "java", "go"},
	"grpc":          {"protobuf", "python", "go", "java", "javascript", "csharp"},
	"kafka":         {"java", "python", "go", "javascript", "scala"},
	"rabbitmq":      {"python", "java", "javascript", "go", "ruby"},
	"stripe":        {"python", "javascript", "ruby", "java", "go", "php", "curl"},
}

// languageCategories groups language identifiers into higher-level categories
// for detecting cross-contamination between unrelated language families.
var languageCategories = map[string]map[string]bool{
	"shell":      {"bash": true, "shell": true, "sh": true, "zsh": true, "fish": true, "powershell": true, "cmd": true, "bat": true},
	"javascript": {"javascript": true, "js": true, "typescript": true, "ts": true, "jsx": true, "tsx": true, "node": true},
	"python":     {"python": true, "py": true, "python3": true},
	"java":       {"java": true, "kotlin": true, "scala": true, "groovy": true},
	"systems":    {"c": true, "cpp": true, "c++": true, "rust": true, "go": true, "zig": true},
	"ruby":       {"ruby": true, "rb": true},
	"dotnet":     {"csharp": true, "cs": true, "fsharp": true, "vb": true},
	"config":     {"yaml": true, "yml": true, "json": true, "toml": true, "ini": true, "xml": true, "hcl": true},
	"query":      {"sql": true, "graphql": true, "cypher": true, "sparql": true},
	"markup":     {"html": true, "css": true, "scss": true, "sass": true, "less": true, "markdown": true, "md": true},
	"mobile":     {"swift": true, "kotlin": true, "dart": true, "objective-c": true, "objc": true},
}

// techPatterns maps framework and runtime names to their language category,
// used to detect technology references in prose that broaden scope.
var techPatterns = map[string]string{
	"node.js": "javascript",
	"react":   "javascript",
	"express": "javascript",
	"django":  "python",
	"flask":   "python",
	"fastapi": "python",
	"spring":  "java",
	"rails":   "ruby",
	"asp.net": "dotnet",
	".net":    "dotnet",
	"swift":   "mobile",
	"flutter": "mobile",
}

// applicationCategories lists language categories classified as application languages.
// These have high syntactic confusion risk with each other (per PLC research).
var applicationCategories = map[string]bool{
	"javascript": true,
	"python":     true,
	"java":       true,
	"systems":    true,
	"ruby":       true,
	"dotnet":     true,
	"mobile":     true,
}

// auxiliaryCategories lists language categories classified as auxiliary (config,
// scripting, markup). These have low confusion risk with application languages.
var auxiliaryCategories = map[string]bool{
	"shell":  true,
	"config": true,
	"query":  true,
	"markup": true,
}

// mismatchWeight returns the similarity weight for a pair of language categories.
// Application↔Application: 1.0 (high syntactic confusion risk)
// Application↔Auxiliary: 0.25 (low confusion risk, syntactically very different)
// Auxiliary↔Auxiliary: 0.1 (minimal confusion risk)
func mismatchWeight(cat1, cat2 string) float64 {
	app1 := applicationCategories[cat1]
	app2 := applicationCategories[cat2]
	if app1 && app2 {
		return 1.0
	}
	aux1 := auxiliaryCategories[cat1]
	aux2 := auxiliaryCategories[cat2]
	if aux1 && aux2 {
		return 0.1
	}
	if (app1 && aux2) || (aux1 && app2) {
		return 0.25
	}
	// Unknown category: treat as application-level mismatch
	return 1.0
}

// Analyze computes contamination metrics for a skill.
// name is the skill name, content is the SKILL.md content,
// codeLanguages are the language identifiers extracted from code blocks.
func Analyze(name, content string, codeLanguages []string) *types.ContaminationReport {
	if codeLanguages == nil {
		codeLanguages = []string{}
	}

	// Detect multi-interface tools
	multiTools := detectMultiInterfaceTools(name, content)

	// Analyze code block language diversity
	langCategories := getLanguageCategories(codeLanguages)

	// Detect additional technology references
	techRefs := detectTechnologyReferences(content)

	// Combine all scope indicators
	allScopes := make(map[string]bool)
	for cat := range langCategories {
		allScopes[cat] = true
	}
	for cat := range techRefs {
		allScopes[cat] = true
	}
	scopeBreadth := len(allScopes)

	// Detect language mismatch
	primaryCategory := findPrimaryCategory(codeLanguages)
	mismatchedCategories := make(map[string]bool)
	if primaryCategory != "" {
		for cat := range langCategories {
			if cat != primaryCategory {
				mismatchedCategories[cat] = true
			}
		}
	}
	languageMismatch := len(mismatchedCategories) > 0

	// Calculate contamination score
	factors := 0.0

	// Factor 1: Multi-interface tool (0.0 or 0.3)
	if len(multiTools) > 0 {
		factors += 0.3
	}

	// Factor 2: Language mismatch in code blocks (0.0-0.4)
	// Weight mismatches by syntactic similarity: application↔application mismatches
	// score higher than application↔auxiliary (per PLC research on language confusion).
	mismatchWeights := make(map[string]float64)
	if languageMismatch {
		weightedMismatch := 0.0
		for cat := range mismatchedCategories {
			w := mismatchWeight(primaryCategory, cat)
			mismatchWeights[cat] = w
			weightedMismatch += w
		}
		mismatchSeverity := math.Min(weightedMismatch/3.0, 1.0)
		factors += 0.4 * mismatchSeverity
	}

	// Factor 3: Scope breadth (0.0-0.3)
	if scopeBreadth > 2 {
		breadthScore := math.Min(float64(scopeBreadth-2)/4.0, 1.0)
		factors += 0.3 * breadthScore
	}

	score := util.RoundTo(math.Min(factors, 1.0), 4)

	// Contamination level
	level := "low"
	if score >= 0.5 {
		level = "high"
	} else if score >= 0.2 {
		level = "medium"
	}

	return &types.ContaminationReport{
		MultiInterfaceTools:  multiTools,
		CodeLanguages:        codeLanguages,
		LanguageCategories:   util.SortedKeys(langCategories),
		PrimaryCategory:      primaryCategory,
		MismatchedCategories: util.SortedKeys(mismatchedCategories),
		MismatchWeights:      mismatchWeights,
		LanguageMismatch:     languageMismatch,
		TechReferences:       util.SortedKeys(techRefs),
		ScopeBreadth:         scopeBreadth,
		ContaminationScore:   score,
		ContaminationLevel:   level,
	}
}

func detectMultiInterfaceTools(name, content string) []string {
	matches := make([]string, 0)
	nameLower := strings.ToLower(name)
	contentLower := strings.ToLower(content)

	for tool := range multiInterfaceTools {
		if strings.Contains(nameLower, tool) || strings.Contains(contentLower, tool) {
			matches = append(matches, tool)
		}
	}
	sort.Strings(matches)
	return matches
}

func getLanguageCategories(languages []string) map[string]bool {
	categories := make(map[string]bool)
	for _, lang := range languages {
		langLower := strings.ToLower(lang)
		for category, members := range languageCategories {
			if members[langLower] {
				categories[category] = true
				break
			}
		}
	}
	return categories
}

func detectTechnologyReferences(content string) map[string]bool {
	refs := make(map[string]bool)
	contentLower := strings.ToLower(content)

	for tech, category := range techPatterns {
		if strings.Contains(contentLower, tech) {
			refs[category] = true
		}
	}

	return refs
}

func findPrimaryCategory(codeLanguages []string) string {
	if len(codeLanguages) == 0 {
		return ""
	}

	counts := make(map[string]int)
	// Track insertion order for tie-breaking (match Python Counter behavior)
	var order []string
	seen := make(map[string]bool)
	for _, lang := range codeLanguages {
		langLower := strings.ToLower(lang)
		for category, members := range languageCategories {
			if members[langLower] {
				counts[category]++
				if !seen[category] {
					seen[category] = true
					order = append(order, category)
				}
				break
			}
		}
	}

	if len(counts) == 0 {
		return ""
	}

	// Find most common category; ties broken by first-encountered order
	maxCount := 0
	primary := ""
	for _, cat := range order {
		if counts[cat] > maxCount {
			maxCount = counts[cat]
			primary = cat
		}
	}
	return primary
}

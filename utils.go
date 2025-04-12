package semchunk

import (
	"regexp"
	"unicode"
)

// LookbehindSplit splits a string at a given splitter, but only if it is preceded by a given string
// This is a helper function to emulate lookbehind in regex
func LookbehindSplit(text string, precededBy string, splitter string) []string {
	escapedPreceder := regexp.QuoteMeta(precededBy)
	escapedSplitter := regexp.QuoteMeta(splitter)
	re := regexp.MustCompile(escapedPreceder + escapedSplitter)
	matches := re.FindAllStringIndex(text, -1)
	parts := make([]string, 0)
	lastIndex := 0
	for _, match := range matches {
		parts = append(parts, text[lastIndex:match[0]]+precededBy)
		lastIndex = match[1]
	}
	parts = append(parts, text[lastIndex:])
	return parts
}

// IsChinese checks if a string is Chinese
func IsChinese(text string) bool {
	if len(text) == 0 {
		return false
	}

	chineseCount := 0
	totalCount := 0

	for _, r := range text {
		// Skip whitespace and punctuation
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}

		totalCount++

		// Check if character is in Chinese Unicode ranges
		// CJK Unified Ideographs (4E00-9FFF)
		// CJK Unified Ideographs Extension A (3400-4DBF)
		// CJK Unified Ideographs Extension B (20000-2A6DF)
		// CJK Unified Ideographs Extension C (2A700-2B73F)
		// CJK Unified Ideographs Extension D (2B740-2B81F)
		// CJK Unified Ideographs Extension E (2B820-2CEAF)
		if (r >= 0x4E00 && r <= 0x9FFF) ||
			(r >= 0x3400 && r <= 0x4DBF) ||
			(r >= 0x20000 && r <= 0x2A6DF) ||
			(r >= 0x2A700 && r <= 0x2B73F) ||
			(r >= 0x2B740 && r <= 0x2B81F) ||
			(r >= 0x2B820 && r <= 0x2CEAF) {
			chineseCount++
		}
	}

	// Consider text as Chinese if more than 50% of characters are Chinese
	return totalCount > 0 && float64(chineseCount)/float64(totalCount) > 0.4
}

func GuessIsChinese(text string, n int) bool {
	if n > len(text) || n <= 0 {
		n = len(text)
	}
	return IsChinese(text[:n])
}

func ContainsSpace(text string) bool {
	for _, r := range text {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

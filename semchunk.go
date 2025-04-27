package semchunk

import (
	"fmt"
	"regexp"
	"strings"
)

// TextSplitter handles the semantic chunking of text
type TextSplitter struct {
	chunkSize      int
	countTokenFunc func(text string) int
	overlap        int
	opts           *TextSplitterOption
}

type TextSplitterOption struct {
	PreserveURLs     bool
	PreservePatterns []*regexp.Regexp
}

func WithPreserveURLs(preserveURLs bool) func(*TextSplitterOption) {
	return func(opts *TextSplitterOption) {
		if opts == nil {
			opts = &TextSplitterOption{}
		}
		opts.PreserveURLs = preserveURLs
		opts.PreservePatterns = append(opts.PreservePatterns, urlRegex)
	}
}

func WithPreservePatterns(preservePatterns ...string) func(*TextSplitterOption) {
	return func(opts *TextSplitterOption) {
		if opts == nil {
			opts = &TextSplitterOption{}
		}
		for _, pattern := range preservePatterns {
			escapedPattern := regexp.QuoteMeta(pattern)
			opts.PreservePatterns = append(opts.PreservePatterns, regexp.MustCompile(escapedPattern))
		}
	}
}

// NewTextSplitter creates a new TextSplitter instance
func NewTextSplitter[K int | float32](chunkSize int, overlap K, countTokenFunc func(text string) int, opts ...func(*TextSplitterOption)) (*TextSplitter, error) {
	var overlapInt int
	if overlapFloat, ok := any(overlap).(float32); ok {
		if overlapFloat < 0 || overlapFloat > 1 {
			return nil, fmt.Errorf("overlap must be between 0 and 1")
		}
		overlapInt = int(overlapFloat * float32(chunkSize))
	} else if overlapInt, ok := any(overlap).(int); ok {
		if overlapInt < 0 || overlapInt > chunkSize {
			return nil, fmt.Errorf("overlap must be between 0 and chunkSize")
		}
	}

	ts := &TextSplitter{
		chunkSize:      chunkSize,
		countTokenFunc: countTokenFunc,
		overlap:        overlapInt,
		opts:           &TextSplitterOption{},
	}

	for _, opt := range opts {
		opt(ts.opts)
	}

	return ts, nil
}

var urlRegex = regexp.MustCompile(`(https?|ftp|file|www)(:|.)(//)?[-A-Za-z0-9+&@#/%?=~_|!:,.;]+[-A-Za-z0-9+&@#/%=~_|]`)
var whitespaceRegex = regexp.MustCompile(`\s+`)
var fullWidthSentenceTerminators = []string{
	"。", "？", "！",
}
var fullWidthClauseSparators = []string{
	"，", "；", "、", "：", "－",
}

var sentenceTerminators = []string{
	".", "?", "!",
}

var clauseSeparators = []string{
	"—", "…", ",", ";", ":",
}

// nonWhitespaceSemanticSplitters defines the splitters in order of preference
var nonWhitespaceSemanticSplitters = append(sentenceTerminators, clauseSeparators...)
var fullWidthNonWhitespaceSemanticSpliters = append(fullWidthSentenceTerminators, fullWidthClauseSparators...)

func longestSplitter(splitters []string) string {
	if len(splitters) == 0 {
		return ""
	}

	longest := splitters[0]
	for _, splitter := range splitters[1:] {
		if len(splitter) > len(longest) {
			longest = splitter
		}
	}
	return longest
}

// innerSplit splits text using the most semantically meaningful splitter possible
func innerSplit(text string, preservePatterns []*regexp.Regexp) (string, bool, []string) {
	splitterIsWhitespace := true

	// Try splitting at newlines
	if strings.Contains(text, "\n") || strings.Contains(text, "\r") {
		re := regexp.MustCompile(`[\r\n]+`)
		matches := re.FindAllString(text, -1)
		if len(matches) > 0 {
			// Find the longest consecutive newlines
			splitter := longestSplitter(matches)
			return splitter, splitterIsWhitespace, strings.Split(text, splitter)
		}
	}

	// Try splitting at tabs
	if strings.Contains(text, "\t") {
		re := regexp.MustCompile(`\t+`)
		matches := re.FindAllString(text, -1)
		if len(matches) > 0 {
			splitter := longestSplitter(matches)
			return splitter, splitterIsWhitespace, strings.Split(text, splitter)
		}
	}

	// Check preserve patterns if they exist
	// if any of the preservePatterns are found, split around them to keep the pattern intact
	for _, pattern := range preservePatterns {
		matches := pattern.FindAllStringIndex(text, -1)
		if len(matches) > 0 {
			// Split the text while keeping the pattern
			parts := make([]string, 0)
			lastIndex := 0
			for _, match := range matches {
				start, end := match[0], match[1]

				// Add the text before the pattern
				if start > lastIndex {
					parts = append(parts, text[lastIndex:start])
				}

				// Add the pattern itself
				parts = append(parts, text[start:end])

				lastIndex = end
			}

			// Add any remaining text
			if lastIndex < len(text) {
				parts = append(parts, text[lastIndex:])
			}

			return "", splitterIsWhitespace, parts
		}
	}

	for _, splitter := range fullWidthNonWhitespaceSemanticSpliters {
		if strings.Contains(text, splitter) {
			splitterIsWhitespace = false
			return splitter, splitterIsWhitespace, strings.Split(text, splitter)
		}
	}

	// Try splitting at whitespace
	if ContainsSpace(text) {
		matches := whitespaceRegex.FindAllString(text, -1)
		if len(matches) > 0 {
			splitter := longestSplitter(matches)

			// If splitter is single character, try to find whitespace preceded by semantic splitters
			if len(splitter) == 1 {
				for _, preceder := range nonWhitespaceSemanticSplitters {
					escapedPreceder := regexp.QuoteMeta(preceder)
					re := regexp.MustCompile(escapedPreceder + `(\s)`)
					if matches := re.FindStringSubmatch(text); matches != nil {
						splitter = matches[1]
						parts := LookbehindSplit(text, preceder, splitter)
						return splitter, splitterIsWhitespace, parts
					}
				}
			}

			return splitter, splitterIsWhitespace, strings.Split(text, splitter)
		}
	}

	// Try non-whitespace semantic splitters
	for _, splitter := range nonWhitespaceSemanticSplitters {
		if strings.Contains(text, splitter) {
			splitterIsWhitespace = false
			return splitter, splitterIsWhitespace, strings.Split(text, splitter)
		}
	}

	// If no semantic splitter found, split into characters
	return "", splitterIsWhitespace, strings.Split(text, "")
}

func estimateSize(size int, splitSize int, splitterSize int, appendSplitter bool) int {
	nextSize := size + splitSize
	if appendSplitter {
		nextSize += splitterSize
	}
	return nextSize
}

// mergeSplits merges splits until a chunk size is reached
func (c *TextSplitter) mergeSplits(splits []string, splitSizes []int, splitIds []int, splitter string, chunkSize int) []string {
	merges := make([]string, 0)
	toMerge := make([]string, 0)
	splitterSize := c.countTokenFunc(splitter)

	size := 0
	for i, split := range splits {
		l := splitSizes[i]

		if estimateSize(size, l, splitterSize, len(toMerge) > 0) > chunkSize {
			merged := strings.Join(toMerge, splitter)
			if len(merged) > 0 {
				merges = append(merges, merged)
			}

			if c.overlap > 0 {
				// keeps popping from the front of toMerge until the size is less than the overlap
				for size > c.overlap || estimateSize(size, l, splitterSize, len(toMerge) > 0) > chunkSize && size > 0 {
					size -= splitSizes[splitIds[0]]
					if len(toMerge) > 1 {
						size -= splitterSize
					}
					toMerge = toMerge[1:]
					splitIds = splitIds[1:]
				}
			} else {
				toMerge = make([]string, 0)
				size = 0
			}
		}

		// still have a chace that single split exceeds chunkSize
		toMerge = append(toMerge, split)
		size += l
		if len(toMerge) > 1 {
			size += splitterSize
		}
	}
	if len(toMerge) > 0 {
		merged := strings.Join(toMerge, splitter)
		if len(merged) > 0 {
			merges = append(merges, merged)
		}
	}

	return merges
}

func (c *TextSplitter) split(text string, chunkSize int, recursionDepth int) []string {
	rets := make([]string, 0)

	splitter, _, splits := innerSplit(text, c.opts.PreservePatterns)

	goodSplits := make([]string, 0)
	goodSplitSizes := make([]int, 0)
	goodSplitIds := make([]int, 0)

	for i, split := range splits {
		l := c.countTokenFunc(split)
		if l < chunkSize {
			goodSplits = append(goodSplits, split)
			goodSplitSizes = append(goodSplitSizes, l)
			goodSplitIds = append(goodSplitIds, i)
			continue
		}
		if len(goodSplits) > 0 {
			merges := c.mergeSplits(goodSplits, goodSplitSizes, goodSplitIds, splitter, chunkSize)

			rets = append(rets, merges...)
			goodSplits = make([]string, 0)
			goodSplitSizes = make([]int, 0)
			goodSplitIds = make([]int, 0)
		}

		newSplits := c.split(split, chunkSize, recursionDepth+1)
		rets = append(rets, newSplits...)
	}

	if len(goodSplits) > 0 {
		merges := c.mergeSplits(goodSplits, goodSplitSizes, goodSplitIds, splitter, chunkSize)
		rets = append(rets, merges...)
	}

	return rets
}

func (c *TextSplitter) Split(text string) []string {
	return c.split(text, c.chunkSize, 0)
}

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
}

// NewTextSplitter creates a new TextSplitter instance
func NewTextSplitter[K int | float32](chunkSize int, overlap K, countTokenFunc func(text string) int) (*TextSplitter, error) {
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

	return &TextSplitter{
		chunkSize:      chunkSize,
		countTokenFunc: countTokenFunc,
		overlap:        overlapInt,
	}, nil
}

var whitespaceRegex = regexp.MustCompile(`\s+`)
var sentenceTerminators = []string{
	"。", "？", "！", ".", "?", "!",
}

var clauseSeparators = []string{
	"，", "；", "：", "—", "－", "…", ",", ";", ":",
}

// nonWhitespaceSemanticSplitters defines the splitters in order of preference
var nonWhitespaceSemanticSplitters = append(sentenceTerminators, clauseSeparators...)

func longest(splitters []string) string {
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
func innerSplit(text string) (string, bool, []string) {
	splitterIsWhitespace := true

	// Try splitting at newlines
	if strings.Contains(text, "\n") || strings.Contains(text, "\r") {
		re := regexp.MustCompile(`[\r\n]+`)
		matches := re.FindAllString(text, -1)
		if len(matches) > 0 {
			// Find the longest consecutive newlines
			splitter := longest(matches)
			return splitter, splitterIsWhitespace, strings.Split(text, splitter)
		}
	}

	// Try splitting at tabs
	if strings.Contains(text, "\t") {
		re := regexp.MustCompile(`\t+`)
		matches := re.FindAllString(text, -1)
		if len(matches) > 0 {
			splitter := longest(matches)
			return splitter, splitterIsWhitespace, strings.Split(text, splitter)
		}
	}

	// Try splitting at whitespace
	if ContainsSpace(text) {
		n := 200
		if n > len(text) {
			n = len(text)
		}
		if !IsChinese(text[:n]) {
			matches := whitespaceRegex.FindAllString(text, -1)
			if len(matches) > 0 {
				splitter := longest(matches)

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

func nextSize(size int, splitLen int, splitterLen int, appendSplit bool) int {
	nextSize := size + splitLen
	if appendSplit {
		nextSize += splitterLen
	}
	return nextSize
}

// mergeSplits merges splits until a chunk size is reached
func (c *TextSplitter) mergeSplits(splits []string, splitLens []int, splitIds []int, splitter string, chunkSize int) []string {
	merges := make([]string, 0)
	toMerge := make([]string, 0)
	splitterLen := c.countTokenFunc(splitter)

	size := 0
	for i, split := range splits {
		l := splitLens[i]

		if nextSize(size, l, splitterLen, len(toMerge) > 0) > chunkSize {
			merged := strings.Join(toMerge, splitter)
			if len(merged) > 0 {
				merges = append(merges, merged)
			}

			if c.overlap > 0 {
				// keeps popping from the front of toMerge until the size is less than the overlap
				for size > c.overlap || nextSize(size, l, splitterLen, len(toMerge) > 0) > chunkSize && size > 0 {
					size -= splitLens[splitIds[0]]
					if len(toMerge) > 1 {
						size -= splitterLen
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

	splitter, _, splits := innerSplit(text)

	goodSplits := make([]string, 0)
	goodSplitLens := make([]int, 0)
	goodSplitIds := make([]int, 0)

	for i, split := range splits {
		l := c.countTokenFunc(split)
		if l < chunkSize {
			goodSplits = append(goodSplits, split)
			goodSplitLens = append(goodSplitLens, l)
			goodSplitIds = append(goodSplitIds, i)
			continue
		}
		if len(goodSplits) > 0 {
			merges := c.mergeSplits(goodSplits, goodSplitLens, goodSplitIds, splitter, chunkSize)

			rets = append(rets, merges...)
			goodSplits = make([]string, 0)
			goodSplitLens = make([]int, 0)
			goodSplitIds = make([]int, 0)
		}

		newSplits := c.split(split, chunkSize, recursionDepth+1)
		rets = append(rets, newSplits...)
	}

	if len(goodSplits) > 0 {
		merges := c.mergeSplits(goodSplits, goodSplitLens, goodSplitIds, splitter, chunkSize)
		rets = append(rets, merges...)
	}

	return rets
}

func (c *TextSplitter) Split(text string) []string {
	return c.split(text, c.chunkSize, 0)
}

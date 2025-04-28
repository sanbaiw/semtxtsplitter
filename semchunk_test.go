package semchunk

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// SimpleTokenCounter implements TokenCounter interface for testing
type SimpleTokenCounter struct {
	wordsPerToken int
}

func NewSimpleTokenCounter(wordsPerToken int) *SimpleTokenCounter {
	return &SimpleTokenCounter{
		wordsPerToken: wordsPerToken,
	}
}

func (c *SimpleTokenCounter) CountTokens(text string) int {
	words := strings.Fields(text)
	return len(words) * c.wordsPerToken
}

func TestInnerSplit(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		splitter     string
		isWhitespace bool
		want         []string
	}{
		{
			name:         "Simple sentence",
			text:         "This is a test sentence.",
			splitter:     " ",
			isWhitespace: true,
			want:         []string{"This", "is", "a", "test", "sentence."},
		},
		{
			name:         "Multiple sentences",
			text:         "First sentence. Second sentence. Third sentence.",
			splitter:     " ",
			isWhitespace: true,
			want:         []string{"First sentence.", "Second sentence.", "Third sentence."},
		},
		{
			name:         "Chinese sentence",
			text:         "文字识别（Optical Character Recognition，OCR）基于腾讯优图实验室的深度学习技术，将图片上的文字内容，智能识别成为可编辑的文本。OCR 支持身份证、名片等卡证类和票据类的印刷体识别，也支持运单等手写体识别，支持提供定制化服务，可以有效地代替人工录入信息。",
			splitter:     "。",
			isWhitespace: false,
			want:         []string{"文字识别（Optical Character Recognition，OCR）基于腾讯优图实验室的深度学习技术，将图片上的文字内容，智能识别成为可编辑的文本", "OCR 支持身份证、名片等卡证类和票据类的印刷体识别，也支持运单等手写体识别，支持提供定制化服务，可以有效地代替人工录入信息", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			splitter, isWhitespace, splits := innerSplit(tt.text, nil)

			assert.Equal(t, tt.splitter, splitter, "%s splitter mismatch", tt.name)
			assert.Equal(t, tt.isWhitespace, isWhitespace, "%s isWhitespace mismatch", tt.name)
			assert.Equal(t, tt.want, splits, "%s splits mismatch", tt.name)
		})
	}
}

func TestMergeSplits(t *testing.T) {
	tests := []struct {
		name      string
		splits    []string
		splitLens []int
		splitIds  []int
		splitter  string
		chunkSize int
		overlap   int
		want      []string
	}{
		{
			name:      "basic merge without overlap",
			splits:    []string{"hello", "world", "test"},
			splitLens: []int{5, 5, 4},
			splitter:  " ",
			chunkSize: 10,
			overlap:   0,
			want:      []string{"hello", "world test"},
		},
		{
			name:      "basic merge without overlap #2",
			splits:    []string{"hello", "world", "test"},
			splitLens: []int{5, 5, 4},
			splitter:  " ",
			chunkSize: 11,
			overlap:   0,
			want:      []string{"hello world", "test"},
		},
		{
			name:      "merge with overlap",
			splits:    []string{"hello", "world", "test", "again"},
			splitLens: []int{5, 5, 4, 5},
			splitter:  " ",
			chunkSize: 11,
			overlap:   5,
			want:      []string{"hello world", "world test", "test again"},
		},
		{
			name:      "single split exceeding chunk size",
			splits:    []string{"thisisaverylongword"},
			splitLens: []int{18},
			splitter:  " ",
			chunkSize: 10,
			overlap:   0,
			want:      []string{"thisisaverylongword"},
		},
		{
			name:      "empty splits",
			splits:    []string{},
			splitLens: []int{},
			splitter:  " ",
			chunkSize: 10,
			overlap:   0,
			want:      []string{},
		},
		{
			name:      "chinese text with overlap",
			splits:    []string{"你好", "世界", "测试"},
			splitLens: []int{2, 2, 2},
			splitter:  "",
			chunkSize: 4,
			overlap:   2,
			want:      []string{"你好世界", "世界测试"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			splitter := &TextSplitter{
				chunkSize:      tt.chunkSize,
				countTokenFunc: func(text string) int { return len(text) },
				overlap:        tt.overlap,
			}

			got := splitter.mergeSplits(tt.splits, tt.splitLens, tt.splitter, tt.chunkSize)

			assert.Equal(t, len(tt.want), len(got), "case: %q got length mismatch", tt.name)
			assert.Equal(t, tt.want, got, "case: %q got mismatch", tt.name)
		})
	}
}

func TestSplit(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		chunkSize int
		overlap   int
		want      []string
	}{
		{
			name:      "simple sentence",
			text:      "This is a test sentence.",
			chunkSize: 10,
			overlap:   0,
			want:      []string{"This is a test sentence."},
		},
		{
			name:      "simple sentence #2",
			text:      "This is a test sentence. This is another test sentence.",
			chunkSize: 5,
			overlap:   0,
			want:      []string{"This is", "a test", "sentence.", "This is", "another test", "sentence."},
		},
		{
			name:      "simple sentence with overlap",
			text:      "This is a test sentence. This is another test sentence.",
			chunkSize: 5,
			overlap:   2,
			want:      []string{"This is", "is a", "a test", "test sentence.", "This is", "is another", "another test", "test sentence."},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			splitter := &TextSplitter{
				chunkSize: tt.chunkSize,
				countTokenFunc: func(text string) int {
					words := strings.Fields(text)
					return len(words) * 2
				},
				overlap: tt.overlap,
				opts:    &TextSplitterOption{},
			}

			got := splitter.Split(tt.text)

			assert.Equal(t, tt.want, got, "case: %q got mismatch", tt.name)
		})
	}
}

func TestInnerSplitWithPreserveURLs(t *testing.T) {
	tests := []struct {
		name         string
		text         string
		splitter     string
		isWhitespace bool
		want         []string
	}{
		{
			name:         "sentence with url",
			text:         "想要与 Claude 聊天？请访问 https://docs.anthropic.com/zh-CN/docs/welcome 如果你是 Claude 新手，从这里开始学习基础知识并进行首次 API 调用。",
			want:         []string{"想要与 Claude 聊天？请访问 ", "https://docs.anthropic.com/zh-CN/docs/welcome", " 如果你是 Claude 新手，从这里开始学习基础知识并进行首次 API 调用。"},
			splitter:     "",
			isWhitespace: true,
		},
		{
			name:         "sentence with starting url",
			text:         "https://docs.anthropic.com/zh-CN/docs/welcome 如果你是 Claude 新手，从这里开始学习基础知识并进行首次 API 调用。",
			want:         []string{"https://docs.anthropic.com/zh-CN/docs/welcome", " 如果你是 Claude 新手，从这里开始学习基础知识并进行首次 API 调用。"},
			splitter:     "",
			isWhitespace: true,
		},
		{
			name:         "sentence with ending url",
			text:         "如果你是 Claude 新手，从这里开始学习基础知识并进行首次 API 调用。https://docs.anthropic.com/zh-CN/docs/welcome",
			want:         []string{"如果你是 Claude 新手，从这里开始学习基础知识并进行首次 API 调用。", "https://docs.anthropic.com/zh-CN/docs/welcome"},
			splitter:     "",
			isWhitespace: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			splitter, isWhitespace, splits := innerSplit(tt.text, []*regexp.Regexp{urlRegex})

			assert.Equal(t, tt.splitter, splitter, "%s splitter mismatch", tt.name)
			assert.Equal(t, tt.isWhitespace, isWhitespace, "%s isWhitespace mismatch", tt.name)
			assert.Equal(t, tt.want, splits, "%s splits mismatch", tt.name)
		})
	}

}

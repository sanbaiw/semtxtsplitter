# Semantic Text Splitter

A Go package for intelligently splitting text into chunks while preserving semantic meaning. This package is particularly useful for processing large texts for language models and other NLP applications.

## Features

- Configurable chunk size
- Support for overlapping chunks
- Customizable token counting
- Semantic-aware text splitting

## Installation

```bash
go get github.com/sanbaiw/semtxtsplitter
```

## Usage
To use the package, you need to provide a token counter function. The following example uses tiktoken to count tokens.

```go
import (
	"log"

	"github.com/pkoukk/tiktoken-go"
	"github.com/sanbaiw/semtxtsplitter"
)

// use tiktoken 
tokenizer, err = tiktoken.GetEncoding("cl100k_base")
if err != nil {
    log.Fatal(err)
}

func tokenCounter(text string) int {
    return len(tokenizer.Encode(text, nil, nil))
}

// Create a text splitter with chunk size of 1000 and 10% overlap
splitter, err := semchunk.NewTextSplitter(1000, 0.1, tokenCounter)
if err != nil {
    log.Fatal(err)
}

// Split your text
splits := splitter.Split(yourText)
```

You can also use other token counters if you prefer.

## License

MIT

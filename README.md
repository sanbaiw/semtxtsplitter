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

```go
import "github.com/sanbaiw/semtxtsplitter"

// Create a text splitter with chunk size of 1000 and 10% overlap
splitter, err := semchunk.NewTextSplitter(1000, 0.1, yourTokenCounter)
if err != nil {
    log.Fatal(err)
}

// Split your text
chunks := splitter.Split(yourText)
```

## License

MIT

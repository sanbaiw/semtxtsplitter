package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	semchunk "github.com/sanbaiw/semtxtsplitter"
)

func main() {
	// Define command line flags
	chunkSize := flag.Int("chunk-size", 100, "Maximum number of tokens per chunk")
	overlap := flag.Float64("overlap", 0.1, "Overlap ratio between chunks (0-1)")
	preserveURLs := flag.Bool("preserve-urls", true, "Preserve URLs in chunks")
	preservePatterns := flag.String("preserve-patterns", "", "Comma-separated list of patterns to preserve")
	flag.Parse()

	// Get input text from arguments or stdin
	var text string
	if len(flag.Args()) > 0 {
		text = strings.Join(flag.Args(), " ")
	} else {
		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		var builder strings.Builder
		for {
			line, err := reader.ReadString('\n')
			builder.WriteString(line)
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
				os.Exit(1)
			}
		}
		text = builder.String()
		if text == "" {
			fmt.Println("Error: No input text provided")
			flag.Usage()
			os.Exit(1)
		}
	}

	// Create token counter function (simple word count for demonstration)
	countTokens := func(text string) int {
		return len(strings.Fields(text))
	}

	// Create options
	var opts []func(*semchunk.TextSplitterOption)
	if *preserveURLs {
		opts = append(opts, semchunk.WithPreserveURLs(true))
	}
	if *preservePatterns != "" {
		patterns := strings.Split(*preservePatterns, ",")
		opts = append(opts, semchunk.WithPreservePatterns(patterns...))
	}

	// Create text splitter
	splitter, err := semchunk.NewTextSplitter(*chunkSize, float32(*overlap), countTokens, opts...)
	if err != nil {
		fmt.Printf("Error creating text splitter: %v\n", err)
		os.Exit(1)
	}

	// Split the text
	chunks := splitter.Split(text)

	// Print results
	fmt.Printf("Input text: %s\n\n", text)
	fmt.Printf("Split into %d chunks:\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("Chunk %d (%d tokens): %s\n", i+1, countTokens(chunk), chunk)
	}
}

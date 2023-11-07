package callbacks

import (
	"context"
	"strings"

	"github.com/tmc/langchaingo/callbacks"
)

// DefaultKeywords is map of the agents final out prefix keywords.
//
//nolint:all
var DefaultKeywords = []string{"Final Answer:", "Final:", "AI:"}

// nolint:all
type StreamHandler struct {
	callbacks.SimpleHandler
	egress          chan []byte
	Keywords        []string
	LastTokens      string
	KeywordDetected bool
	PrintOutput     bool
}

var _ callbacks.Handler = &StreamHandler{}

func NewStreamHandler(keywords ...string) *StreamHandler {
	if len(keywords) > 0 {
		DefaultKeywords = keywords
	}

	return &StreamHandler{
		egress:   make(chan []byte),
		Keywords: DefaultKeywords,
	}
}

func (handler *StreamHandler) GetEgress() chan []byte {
	return handler.egress
}

func (handler *StreamHandler) ReadFromEgress(ctx context.Context, callback func(ctx context.Context, chunk []byte)) {
	go func() {
		defer close(handler.egress)
		for data := range handler.egress {
			callback(ctx, data)
		}
	}()
}

func (handler *StreamHandler) HandleChainStart(_ context.Context, inputs map[string]any) {
	startString := "[chain start]"
	handler.egress <- []byte(startString)
}

func (handler *StreamHandler) HandleChainEnd(_ context.Context, outputs map[string]any) {
	endString := "[chain end]"
	handler.egress <- []byte(endString)
	handler.egress <- []byte("END")
}

func (handler *StreamHandler) HandleStreamingFunc(ctx context.Context, chunk []byte) {
	chunkStr := string(chunk)
	handler.LastTokens += chunkStr

	// Buffer the last few chunks to match the longest keyword size
	longestSize := len(handler.Keywords[0])
	for _, k := range handler.Keywords {
		if len(k) > longestSize {
			longestSize = len(k)
		}
	}

	if len(handler.LastTokens) > longestSize {
		handler.LastTokens = handler.LastTokens[len(handler.LastTokens)-longestSize:]
	}

	// Check for keywords
	for _, k := range DefaultKeywords {
		if strings.Contains(handler.LastTokens, k) {
			handler.KeywordDetected = true
		}
	}

	// Check for colon and set print mode.
	if handler.KeywordDetected && chunkStr != ":" {
		handler.PrintOutput = true
	}

	// Print the final output after the detection of keyword.
	if handler.PrintOutput {
		handler.egress <- chunk
	}
}
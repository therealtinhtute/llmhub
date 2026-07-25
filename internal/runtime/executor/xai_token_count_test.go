package executor

import (
	"testing"

	"github.com/tiktoken-go/tokenizer"
)

func TestCountXAIInputTokensExcludesRequestStructure(t *testing.T) {
	enc, err := tokenizer.Get(tokenizer.O200kBase)
	if err != nil {
		t.Fatalf("tokenizer.Get() error = %v", err)
	}

	semanticBody := []byte(`{
		"instructions":"Follow the repository instructions.",
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"Review this implementation."}]},
			{"type":"function_call","name":"read_file","arguments":"{\"path\":\"main.go\"}"},
			{"type":"function_call_output","output":"package main"},
			{"type":"reasoning","summary":[{"type":"summary_text","text":"I will inspect the file."}]}
		],
		"tools":[{"type":"function","name":"read_file","description":"Reads a file.","parameters":{"type":"object","properties":{"path":{"type":"string"}}}}],
		"text":{"format":{"name":"result","schema":{"type":"object"}}}
	}`)
	structuralBody := []byte(`{
		"model":"grok-4.5", "stream":false, "reasoning":{"effort":"high"},
		"metadata":{"large_wrapper":"this metadata must not affect estimated input tokens"},
		"prompt_cache_key":"session-123", "max_output_tokens":4096,
		"instructions":"Follow the repository instructions.",
		"input":[
			{"type":"message","role":"user","content":[{"type":"input_text","text":"Review this implementation."}]},
			{"type":"function_call","name":"read_file","arguments":"{\"path\":\"main.go\"}"},
			{"type":"function_call_output","output":"package main"},
			{"type":"reasoning","summary":[{"type":"summary_text","text":"I will inspect the file."}]}
		],
		"tools":[{"type":"function","name":"read_file","description":"Reads a file.","parameters":{"type":"object","properties":{"path":{"type":"string"}}}}],
		"text":{"format":{"name":"result","schema":{"type":"object"}}}
	}`)

	semanticCount, err := countXAIInputTokens(enc, semanticBody)
	if err != nil {
		t.Fatalf("countXAIInputTokens() error = %v", err)
	}
	structuralCount, err := countXAIInputTokens(enc, structuralBody)
	if err != nil {
		t.Fatalf("countXAIInputTokens() error = %v", err)
	}
	if structuralCount != semanticCount {
		t.Fatalf("structural count = %d, want %d", structuralCount, semanticCount)
	}

	for name, tc := range map[string]struct {
		body     []byte
		expected string
	}{
		"instructions": {
			body:     []byte(`{"instructions":"unique instruction text"}`),
			expected: "unique instruction text",
		},
		"string input": {
			body:     []byte(`{"input":"unique input text"}`),
			expected: "unique input text",
		},
		"message content": {
			body:     []byte(`{"input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"unique message text"}]}]}`),
			expected: "unique message text",
		},
		"refusal": {
			body:     []byte(`{"input":[{"type":"message","content":[{"type":"refusal","refusal":"unique refusal text"}]}]}`),
			expected: "unique refusal text",
		},
		"input image": {
			body:     []byte(`{"input":[{"type":"message","content":[{"type":"input_image","image_url":"https://example.com/unique.png"}]}]}`),
			expected: "https://example.com/unique.png",
		},
		"input file": {
			body:     []byte(`{"input":[{"type":"message","content":[{"type":"input_file","file_data":"unique file data","filename":"unique.txt"}]}]}`),
			expected: "unique file data\nunique.txt",
		},
		"input audio": {
			body:     []byte(`{"input":[{"type":"message","content":[{"type":"input_audio","data":"unique audio data"}]}]}`),
			expected: "unique audio data",
		},
		"function call": {
			body:     []byte(`{"input":[{"type":"function_call","call_id":"call-1","name":"unique_function","arguments":"{\"value\":\"unique argument\"}"}]}`),
			expected: "unique_function\n{\"value\":\"unique argument\"}",
		},
		"function call output": {
			body:     []byte(`{"input":[{"type":"function_call_output","call_id":"call-1","output":"unique tool output"}]}`),
			expected: "unique tool output",
		},
		"reasoning summary": {
			body:     []byte(`{"input":[{"type":"reasoning","summary":[{"type":"summary_text","text":"unique summary text"}]}]}`),
			expected: "unique summary text",
		},
		"function tool": {
			body:     []byte(`{"tools":[{"type":"function","name":"unique_tool","description":"unique tool description","parameters":{"type":"object","properties":{"value":{"type":"string"}}}}]}`),
			expected: "unique_tool\nunique tool description\n{\"type\":\"object\",\"properties\":{\"value\":{\"type\":\"string\"}}}",
		},
		"structured text format": {
			body:     []byte(`{"text":{"format":{"name":"unique_format","schema":{"type":"object","properties":{"value":{"type":"string"}}}}}}`),
			expected: "unique_format\n{\"type\":\"object\",\"properties\":{\"value\":{\"type\":\"string\"}}}",
		},
	} {
		t.Run(name, func(t *testing.T) {
			count, errCount := countXAIInputTokens(enc, tc.body)
			if errCount != nil {
				t.Fatalf("countXAIInputTokens() error = %v", errCount)
			}
			expected, errExpected := enc.Count(tc.expected)
			if errExpected != nil {
				t.Fatalf("encoder.Count() error = %v", errExpected)
			}
			if count != int64(expected) {
				t.Fatalf("countXAIInputTokens() = %d, want %d", count, expected)
			}
		})
	}
}

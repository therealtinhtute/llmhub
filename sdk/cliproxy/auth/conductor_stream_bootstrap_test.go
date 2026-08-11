package auth

// Gate test for Plan D (model-combos): a failed streaming candidate can only be
// replaced while nothing has been flushed to the client. These tests pin that
// streamBootstrapError is returned only when zero chunks were written downstream,
// and that a mid-stream upstream failure produces a different error type.
// See docs/plans/active/model-combos.md ## Load-bearing assumption.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	cliproxyexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

func TestStreamBootstrapErrorOnlyBeforeFirstChunk(t *testing.T) {
	exec := &openAICompatPoolExecutor{id: "pool"}
	exec.streamFirstErrors = map[string]error{
		"m1": &Error{HTTPStatus: http.StatusInternalServerError, Message: "upstream exploded at bootstrap", Retryable: true},
	}
	manager := NewManager(nil, nil, nil)
	auth := &Auth{ID: "pool-auth", Provider: "pool", Status: StatusActive}

	ctx := context.Background()
	req := cliproxyexecutor.Request{Model: "m1"}
	result, err := manager.executeStreamWithModelPool(ctx, exec, auth, "pool", req, cliproxyexecutor.Options{}, "m1", []string{"m1"}, false)

	var bootstrapErr *streamBootstrapError
	if !errors.As(err, &bootstrapErr) {
		t.Fatalf("expected *streamBootstrapError on bootstrap failure, got %T: %v", err, err)
	}
	if result != nil {
		t.Fatalf("expected nil StreamResult on bootstrap failure, got %+v", result)
	}
	if !strings.Contains(bootstrapErr.Error(), "upstream exploded at bootstrap") {
		t.Fatalf("bootstrap error lost the cause message: %v", bootstrapErr)
	}
	// Zero chunks were written downstream: the error path returns no stream at all.
}

func TestStreamBootstrapMidStreamFailureNotBootstrapped(t *testing.T) {
	midErr := &Error{HTTPStatus: http.StatusInternalServerError, Message: "upstream died mid-stream"}
	exec := &openAICompatPoolExecutor{id: "pool"}
	exec.streamPayloads = map[string][]cliproxyexecutor.StreamChunk{
		"m1": {
			{Payload: []byte("first-chunk")},
			{Err: midErr},
		},
	}
	manager := NewManager(nil, nil, nil)
	auth := &Auth{ID: "pool-auth", Provider: "pool", Status: StatusActive}

	ctx := context.Background()
	req := cliproxyexecutor.Request{Model: "m1"}
	result, err := manager.executeStreamWithModelPool(ctx, exec, auth, "pool", req, cliproxyexecutor.Options{}, "m1", []string{"m1"}, false)

	if err != nil {
		t.Fatalf("mid-stream failure must not be returned as an error, got %T: %v", err, err)
	}
	if result == nil {
		t.Fatal("expected a StreamResult for a mid-stream failure")
	}
	var got []cliproxyexecutor.StreamChunk
	for chunk := range result.Chunks {
		got = append(got, chunk)
	}
	if len(got) < 2 || string(got[0].Payload) != "first-chunk" {
		t.Fatalf("expected the buffered payload chunk first, got %+v", got)
	}
	last := got[len(got)-1]
	if last.Err == nil {
		t.Fatalf("expected the mid-stream error to reach the client in-stream, got %+v", got)
	}
	var bootstrapErr *streamBootstrapError
	if errors.As(last.Err, &bootstrapErr) {
		t.Fatalf("mid-stream failure must not surface as *streamBootstrapError")
	}
	if !strings.Contains(last.Err.Error(), "upstream died mid-stream") {
		t.Fatalf("in-stream error lost its message: %v", last.Err)
	}
	if !errors.Is(last.Err, midErr) {
		t.Fatalf("in-stream error should be the original upstream error value")
	}
}

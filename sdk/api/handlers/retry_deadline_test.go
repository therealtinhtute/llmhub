package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	coreauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	coreexecutor "github.com/therealtinhtute/llmhub/sdk/cliproxy/executor"
)

// Ported from upstream CLIProxyAPI commit 6724a95851f9
// (sdk/api/handlers/retry_deadline_test.go). Local adaptation: model-scoped
// blocking consults ModelStates, so the synthetic credential carries a
// per-model unavailable state in addition to the auth-level flags.
func TestLocalRecoveryHintsSurviveDefaultHeaderPolicyAndEnrichment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, quota := range []bool{false, true} {
		next := time.Now().Add(time.Minute)
		_, err := (&coreauth.RoundRobinSelector{}).Pick(context.Background(), "codex", "gpt-5.6-luna", coreexecutor.Options{}, []*coreauth.Auth{{
			ID: "synthetic", Provider: "codex", Unavailable: true, NextRetryAfter: next,
			Quota: coreauth.QuotaState{Exceeded: quota},
			ModelStates: map[string]*coreauth.ModelState{
				"gpt-5.6-luna": {Unavailable: true, NextRetryAfter: next, Quota: coreauth.QuotaState{Exceeded: quota}},
			},
		}})
		if err == nil {
			t.Fatal("expected unavailable credential")
		}
		err = enrichAuthSelectionError(fmt.Errorf("selection: %w", err), []string{"codex"}, "gpt-5.6-luna")
		for _, streaming := range []bool{false, true} {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
			if streaming {
				ctx.Request.Header.Set("Accept", "text/event-stream")
			}
			NewBaseAPIHandlers(nil, nil).WriteErrorResponse(ctx, executionErrorMessage(err))
			wantStatus := 503
			if quota {
				wantStatus = 429
			}
			if recorder.Code != wantStatus || recorder.Header().Get("Retry-After") != "60" {
				t.Fatalf("quota=%t streaming=%t status=%d Retry-After=%q", quota, streaming, recorder.Code, recorder.Header().Get("Retry-After"))
			}
			if !strings.Contains(recorder.Body.String(), "gpt-5.6-luna") {
				t.Fatal("model context was lost")
			}
		}
	}
}

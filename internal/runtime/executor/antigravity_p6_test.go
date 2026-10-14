package executor

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// Ports upstream d8f2dceef (batched functionResponse name repair) and the
// e44432ab batch + 1c45093d10f6 strict-guard degrade shape, adapted to keep
// local signature semantics.
func TestAntigravityP6RepairAndDegrade(t *testing.T) {
	t.Run("unknown and empty names repaired from functionCall", func(t *testing.T) {
		payload := []byte(`{"request":{"contents":[{"role":"model","parts":[{"functionCall":{"id":"call-1","name":"read","args":{}}}]},{"role":"function","parts":[{"functionResponse":{"id":"call-1","name":"unknown","response":{"ok":true}}},{"functionResponse":{"id":"call-1","name":"","response":{"ok":true}}}]}]}}`)
		out := repairAntigravityGeminiFunctionResponseNames(payload)
		for _, path := range []string{
			"request.contents.1.parts.0.functionResponse.name",
			"request.contents.1.parts.1.functionResponse.name",
		} {
			if got := gjson.GetBytes(out, path).String(); got != "read" {
				t.Fatalf("%s = %q, want read: %s", path, got, out)
			}
		}
		if again := repairAntigravityGeminiFunctionResponseNames(out); string(again) != string(out) {
			t.Fatalf("repair not idempotent:\n%s\n%s", out, again)
		}
	})

	t.Run("good names untouched", func(t *testing.T) {
		good := []byte(`{"request":{"contents":[{"role":"model","parts":[{"functionCall":{"id":"c1","name":"read","args":{}}}]},{"role":"function","parts":[{"functionResponse":{"id":"c1","name":"already_good","response":{}}}]}]}}`)
		if got := repairAntigravityGeminiFunctionResponseNames(good); string(got) != string(good) {
			t.Fatalf("good name changed: %s", got)
		}
	})

	t.Run("normalize wires repair and role fix", func(t *testing.T) {
		in := []byte(`{"request":{"contents":[{"role":"model","parts":[{"functionCall":{"id":"a","name":"actual","args":{}}}]},{"role":"user","parts":[{"functionResponse":{"id":"a","name":"unknown","response":{}}}]}]}}`)
		out := normalizeAntigravityGeminiFunctionResponseRoles(in)
		if got := gjson.GetBytes(out, "request.contents.1.parts.0.functionResponse.name").String(); got != "actual" {
			t.Fatalf("name = %q, want actual: %s", got, out)
		}
		if got := gjson.GetBytes(out, "request.contents.1.role").String(); got != "function" {
			t.Fatalf("role = %q, want function: %s", got, out)
		}
	})

	t.Run("degrade rewrites reserved ids with strict guard", func(t *testing.T) {
		reserved := antigravityGeminiClaudeToolUseID("native-1", "Read", `{"file_path":"/tmp/a"}`)
		if reserved == "" {
			t.Fatal("no reserved id")
		}
		in := []byte(`{"request":{"contents":[{"role":"model","parts":[{"functionCall":{"id":"` + reserved + `","name":"Read","args":{"file_path":"/tmp/a"}}}]},{"role":"user","parts":[{"functionResponse":{"id":"` + reserved + `","name":"Read","response":{"result":"ok"}}}]}]}}`)
		out, n := degradeAntigravityClaudeToolProvenanceIDs(in)
		if n != 2 {
			t.Fatalf("degraded = %d, want 2: %s", n, out)
		}
		if strings.Contains(string(out), reserved) {
			t.Fatalf("reserved id still present: %s", out)
		}
		if synth := antigravitySyntheticToolCallID(reserved); !strings.Contains(string(out), synth) {
			t.Fatalf("synthetic %q missing: %s", synth, out)
		}
	})
}

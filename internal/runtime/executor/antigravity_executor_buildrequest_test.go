package executor

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	cliproxyauth "github.com/therealtinhtute/llmhub/sdk/cliproxy/auth"
	"github.com/tidwall/gjson"
)

func TestAntigravityBuildRequest_SanitizesGeminiToolSchema(t *testing.T) {
	body := buildRequestBodyFromPayload(t, "gemini-2.5-pro")

	decl := extractFirstFunctionDeclaration(t, body)
	if _, ok := decl["parametersJsonSchema"]; ok {
		t.Fatalf("parametersJsonSchema should be renamed to parameters")
	}

	params, ok := decl["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("parameters missing or invalid type")
	}
	assertSchemaSanitizedAndPropertyPreserved(t, params)
}

func TestAntigravityBuildRequest_SanitizesAntigravityToolSchema(t *testing.T) {
	body := buildRequestBodyFromPayload(t, "claude-opus-4-6")

	decl := extractFirstFunctionDeclaration(t, body)
	params, ok := decl["parameters"].(map[string]any)
	if !ok {
		t.Fatalf("parameters missing or invalid type")
	}
	assertSchemaSanitizedAndPropertyPreserved(t, params)
}

func TestAntigravityBuildRequest_SkipsSchemaSanitizationWithoutToolsField(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3.1-flash-image", []byte(`{
		"request": {
			"contents": [
				{
					"role": "user",
					"x-debug": "keep-me",
					"parts": [
						{
							"text": "hello"
						}
					]
				}
			],
			"nonSchema": {
				"nullable": true,
				"x-extra": "keep-me"
			},
			"generationConfig": {
				"maxOutputTokens": 128
			}
		}
	}`))

	assertNonSchemaRequestPreserved(t, body)
}

func TestAntigravityBuildRequest_SkipsSchemaSanitizationWithEmptyToolsArray(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3.1-flash-image", []byte(`{
		"request": {
			"tools": [],
			"contents": [
				{
					"role": "user",
					"x-debug": "keep-me",
					"parts": [
						{
							"text": "hello"
						}
					]
				}
			],
			"nonSchema": {
				"nullable": true,
				"x-extra": "keep-me"
			},
			"generationConfig": {
				"maxOutputTokens": 128
			}
		}
	}`))

	assertNonSchemaRequestPreserved(t, body)
}

func TestAntigravityBuildRequest_PreservesHistoryWhileSanitizingSchemas(t *testing.T) {
	payload := []byte(`{
		"request": {
			"contents": [
				{"role": "model", "parts": [{"functionCall": {"name": "manage_todo_list", "args": {
					"operation": "write",
					"todoList": [
						{"id": 1, "title": "output 1", "description": "d1", "status": "not-started"},
						{"id": 2, "title": "output 2", "description": "d2", "status": "not-started"}
					]
				}}}]},
				{"role": "model", "parts": [{"functionCall": {"name": "write_file", "args": {
					"path": "a.md", "format": "markdown", "default": "x", "pattern": "p",
					"const": "c", "deprecated": false, "nullable": "n", "examples": "e",
					"additionalProperties": "ap", "x-custom": "keepme"
				}}}]}
			],
			"tools": [{"functionDeclarations": [{
				"name": "manage_todo_list",
				"parametersJsonSchema": {
					"type": "object",
					"required": ["todoList"],
					"properties": {"todoList": {"type": "array", "items": {
						"type": "object",
						"required": ["id", "title"],
						"title": "TodoItem",
						"properties": {"id": {"type": "number"}, "title": {"type": "string", "minLength": 3}}
					}}}
				}
			}] }]
		}
	}`)

	body := buildRequestBodyFromRawPayload(t, "gemini-3.1-pro", payload)
	encoded, errMarshal := json.Marshal(body)
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}

	todo := gjson.GetBytes(encoded, `request.contents.0.parts.0.functionCall.args.todoList`)
	for i, item := range todo.Array() {
		if !item.Get("title").Exists() {
			t.Errorf("todoList[%d] lost its title: %s", i, item.Raw)
		}
	}

	args := gjson.GetBytes(encoded, `request.contents.1.parts.0.functionCall.args`)
	for _, key := range []string{"format", "default", "pattern", "const", "deprecated", "examples", "additionalProperties", "x-custom"} {
		if !args.Get(gjson.Escape(key)).Exists() {
			t.Errorf("argument key %q was stripped from history: %s", key, args.Raw)
		}
	}
	if args.Get("enum").Exists() {
		t.Errorf("cleaner fabricated an enum key in history args: %s", args.Raw)
	}

	items := gjson.GetBytes(encoded, "request.tools.0.functionDeclarations.0.parameters.properties.todoList.items")
	if items.Get("title").Exists() {
		t.Errorf("schema keyword title was not removed: %s", items.Raw)
	}
	if items.Get("properties.title.minLength").Exists() {
		t.Errorf("unsupported keyword minLength was not removed: %s", items.Raw)
	}
	if !items.Get("properties.title").Exists() {
		t.Errorf("schema property named title must be preserved: %s", items.Raw)
	}
}

func TestAntigravityBuildRequest_SanitizesAdditionalSchemaLocations(t *testing.T) {
	payload := []byte(`{"request": {
		"tools": [{"function_declarations": [{
			"name": "t",
			"parameters_json_schema": {"type": "object", "$id": "drop-a", "properties": {"a": {"type": "string"}}},
			"response": {"type": "object", "$comment": "drop-b", "properties": {"b": {"type": "string"}}},
			"response_json_schema": {"type": "object", "$id": "drop-c", "properties": {"c": {"type": "string"}}}
		}] }],
		"generation_config": {"response_schema": {"type": "object", "$id": "drop-d", "properties": {"d": {"type": "string"}}}}
	}}`)

	body := buildRequestBodyFromRawPayload(t, "gemini-3.1-pro", payload)
	encoded, errMarshal := json.Marshal(body)
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}

	for _, path := range []string{
		`request.tools.0.function_declarations.0.parameters_json_schema.\$id`,
		`request.tools.0.function_declarations.0.response.\$comment`,
		`request.tools.0.function_declarations.0.response_json_schema.\$id`,
		`request.generation_config.response_schema.\$id`,
	} {
		if gjson.GetBytes(encoded, path).Exists() {
			t.Errorf("unsupported keyword at %s survived cleaning: %s", path, encoded)
		}
	}
}

func TestAntigravityBuildRequest_PreservesResponseUnionAndEnumType(t *testing.T) {
	payload := []byte(`{"request":{
		"tools":[{"functionDeclarations":[{"name":"tool","parameters":{"type":"object","properties":{
			"choice":{"anyOf":[{"type":"string"},{"type":"null"}]},
			"level":{"type":"number","enum":[1,2]}
		}}}]}],
		"generationConfig":{"responseSchema":{"type":"object","properties":{
			"action":{"anyOf":[{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]},{"type":"null"}]},
			"conviction":{"type":"number","enum":[0.25,0.5,1]}
		}}}
	}}`)

	body := buildRequestBodyFromRawPayload(t, "gemini-3.1-pro", payload)
	encoded, errMarshal := json.Marshal(body)
	if errMarshal != nil {
		t.Fatal(errMarshal)
	}

	responseSchema := gjson.GetBytes(encoded, "request.generationConfig.responseSchema")
	union := responseSchema.Get("properties.action.anyOf")
	if !union.IsArray() || len(union.Array()) != 2 || union.Get("1.type").String() != "null" {
		t.Errorf("response anyOf union was flattened: %s", responseSchema.Raw)
	}
	conviction := responseSchema.Get("properties.conviction")
	if gotType := conviction.Get("type").String(); gotType != "number" {
		t.Errorf("response enum type = %q, want number: %s", gotType, responseSchema.Raw)
	}
	for _, enumValue := range conviction.Get("enum").Array() {
		if enumValue.Type != gjson.String {
			t.Errorf("response enum value is not a string: %s", conviction.Raw)
		}
	}

	toolSchema := gjson.GetBytes(encoded, "request.tools.0.functionDeclarations.0.parameters")
	if toolSchema.Get("properties.choice.anyOf").Exists() {
		t.Errorf("tool anyOf union was not flattened: %s", toolSchema.Raw)
	}
	if gotType := toolSchema.Get("properties.level.type").String(); gotType != "string" {
		t.Errorf("tool enum type = %q, want string: %s", gotType, toolSchema.Raw)
	}
}

func TestAntigravityBuildRequest_UsesAuthProjectID(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3.1-pro", []byte(`{
		"request": {
			"contents": [
				{
					"role": "user",
					"parts": [{"text": "hello"}]
				}
			]
		}
	}`))

	if got, ok := body["project"].(string); !ok || got != "project-1" {
		t.Fatalf("project should come from auth metadata, got=%v", body["project"])
	}
}

func TestAntigravityPrepareRequestAuth_FetchesMissingProjectID(t *testing.T) {
	executor := &AntigravityExecutor{}
	auth := &cliproxyauth.Auth{Metadata: map[string]any{
		"access_token": "token",
		"expired":      time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}}
	ctx := context.WithValue(context.Background(), "cliproxy.roundtripper", roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://cloudcode-pa.googleapis.com/v1internal:loadCodeAssist" {
			t.Fatalf("unexpected project discovery request: %s", req.URL.String())
		}
		if got := req.Header.Get("X-Goog-Api-Client"); got != "" {
			t.Fatalf("X-Goog-Api-Client = %q, want empty", got)
		}
		raw, errRead := io.ReadAll(req.Body)
		if errRead != nil {
			t.Fatalf("read discovery body: %v", errRead)
		}
		if !strings.Contains(string(raw), `"ideType":"ANTIGRAVITY"`) {
			t.Fatalf("unexpected discovery body: %s", string(raw))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"cloudaicompanionProject":"fetched-project"}`)),
		}, nil
	}))

	updated, err := executor.PrepareRequestAuth(ctx, auth)
	if err != nil {
		t.Fatalf("PrepareRequestAuth error: %v", err)
	}
	if updated == nil {
		t.Fatalf("PrepareRequestAuth returned nil auth")
	}
	if _, ok := auth.Metadata["project_id"]; ok {
		t.Fatalf("original auth metadata should not be mutated")
	}
	if got, ok := updated.Metadata["project_id"].(string); !ok || got != "fetched-project" {
		t.Fatalf("updated auth metadata project_id = %v, want fetched-project", updated.Metadata["project_id"])
	}
}

func TestAntigravityBuildRequest_RejectsMissingProjectID(t *testing.T) {
	executor := &AntigravityExecutor{}
	auth := &cliproxyauth.Auth{Metadata: map[string]any{}}

	_, err := executor.buildRequest(context.Background(), auth, "token", "gemini-3.1-pro", []byte(`{"request":{}}`), false, "", "https://example.com")
	if err == nil {
		t.Fatalf("buildRequest should fail when auth has no project_id")
	}
	status, ok := err.(interface{ StatusCode() int })
	if !ok {
		t.Fatalf("error should expose status code, got %T", err)
	}
	if got := status.StatusCode(); got != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", got, http.StatusBadRequest)
	}
}

func assertNonSchemaRequestPreserved(t *testing.T, body map[string]any) {
	t.Helper()

	request, ok := body["request"].(map[string]any)
	if !ok {
		t.Fatalf("request missing or invalid type")
	}

	contents, ok := request["contents"].([]any)
	if !ok || len(contents) == 0 {
		t.Fatalf("contents missing or empty")
	}
	content, ok := contents[0].(map[string]any)
	if !ok {
		t.Fatalf("content missing or invalid type")
	}
	if got, ok := content["x-debug"].(string); !ok || got != "keep-me" {
		t.Fatalf("x-debug should be preserved when no tool schema exists, got=%v", content["x-debug"])
	}

	nonSchema, ok := request["nonSchema"].(map[string]any)
	if !ok {
		t.Fatalf("nonSchema missing or invalid type")
	}
	if _, ok := nonSchema["nullable"]; !ok {
		t.Fatalf("nullable should be preserved outside schema cleanup path")
	}
	if got, ok := nonSchema["x-extra"].(string); !ok || got != "keep-me" {
		t.Fatalf("x-extra should be preserved outside schema cleanup path, got=%v", nonSchema["x-extra"])
	}

	if generationConfig, ok := request["generationConfig"].(map[string]any); ok {
		if _, ok := generationConfig["maxOutputTokens"]; ok {
			t.Fatalf("maxOutputTokens should still be removed for non-Claude requests")
		}
	}
}

func buildRequestBodyFromPayload(t *testing.T, modelName string) map[string]any {
	t.Helper()
	return buildRequestBodyFromRawPayload(t, modelName, []byte(`{
		"request": {
			"tools": [
				{
					"function_declarations": [
						{
							"name": "tool_1",
							"parametersJsonSchema": {
								"$schema": "http://json-schema.org/draft-07/schema#",
								"$id": "root-schema",
								"type": "object",
								"properties": {
									"$id": {"type": "string"},
									"arg": {
										"type": "object",
										"prefill": "hello",
										"properties": {
											"mode": {
												"type": "string",
												"deprecated": true,
												"enum": ["a", "b"],
												"enumTitles": ["A", "B"]
											}
										}
									}
								},
								"patternProperties": {
									"^x-": {"type": "string"}
								}
							}
						}
					]
				}
			]
		}
	}`))
}

func buildRequestBodyFromRawPayload(t *testing.T, modelName string, payload []byte) map[string]any {
	t.Helper()

	executor := &AntigravityExecutor{}
	auth := &cliproxyauth.Auth{Metadata: map[string]any{"project_id": "project-1"}}

	req, err := executor.buildRequest(context.Background(), auth, "token", modelName, payload, false, "", "https://example.com")
	if err != nil {
		t.Fatalf("buildRequest error: %v", err)
	}

	return requestBody(t, req)
}

func requestBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()

	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read request body error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal request body error: %v, body=%s", err, string(raw))
	}
	return body
}

func extractFirstFunctionDeclaration(t *testing.T, body map[string]any) map[string]any {
	t.Helper()

	request, ok := body["request"].(map[string]any)
	if !ok {
		t.Fatalf("request missing or invalid type")
	}
	tools, ok := request["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("tools missing or empty")
	}
	tool, ok := tools[0].(map[string]any)
	if !ok {
		t.Fatalf("first tool invalid type")
	}
	decls, ok := tool["function_declarations"].([]any)
	if !ok || len(decls) == 0 {
		t.Fatalf("function_declarations missing or empty")
	}
	decl, ok := decls[0].(map[string]any)
	if !ok {
		t.Fatalf("first function declaration invalid type")
	}
	return decl
}

func assertSchemaSanitizedAndPropertyPreserved(t *testing.T, params map[string]any) {
	t.Helper()

	if _, ok := params["$id"]; ok {
		t.Fatalf("root $id should be removed from schema")
	}
	if _, ok := params["patternProperties"]; ok {
		t.Fatalf("patternProperties should be removed from schema")
	}

	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing or invalid type")
	}
	if _, ok := props["$id"]; !ok {
		t.Fatalf("property named $id should be preserved")
	}

	arg, ok := props["arg"].(map[string]any)
	if !ok {
		t.Fatalf("arg property missing or invalid type")
	}
	if _, ok := arg["prefill"]; ok {
		t.Fatalf("prefill should be removed from nested schema")
	}

	argProps, ok := arg["properties"].(map[string]any)
	if !ok {
		t.Fatalf("arg.properties missing or invalid type")
	}
	mode, ok := argProps["mode"].(map[string]any)
	if !ok {
		t.Fatalf("mode property missing or invalid type")
	}
	if _, ok := mode["enumTitles"]; ok {
		t.Fatalf("enumTitles should be removed from nested schema")
	}
	if _, ok := mode["deprecated"]; ok {
		t.Fatalf("deprecated should be removed from nested schema")
	}
}

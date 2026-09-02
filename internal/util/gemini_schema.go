// Package util provides utility functions for the CLI Proxy API server.
package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var gjsonPathKeyReplacer = strings.NewReplacer(".", "\\.", "*", "\\*", "?", "\\?")

const placeholderReasonDescription = "Brief explanation of why you are calling this tool"

// CleanJSONSchemaForAntigravity transforms a JSON schema to be compatible with Antigravity API.
// It handles unsupported keywords, type flattening, and schema simplification while preserving
// semantic information as description hints.
func CleanJSONSchemaForAntigravity(jsonStr string) string {
	return cleanJSONSchema(jsonStr, true)
}

// CleanJSONSchemaForAntigravityResponse transforms a response schema without applying
// tool-only compatibility rewrites that would alter the client's structured output contract.
func CleanJSONSchemaForAntigravityResponse(jsonStr string) string {
	jsonStr = convertRefsToHints(jsonStr)
	jsonStr = convertConstToEnum(jsonStr)
	jsonStr = convertEnumValuesToStringsPreservingType(jsonStr)
	jsonStr = addEnumHints(jsonStr)
	jsonStr = addAdditionalPropertiesHints(jsonStr)
	jsonStr = moveConstraintsToDescription(jsonStr)
	jsonStr = mergeAllOf(jsonStr)
	jsonStr = flattenTypeArrays(jsonStr)
	jsonStr = removeUnsupportedKeywords(jsonStr)
	jsonStr = cleanupRequiredFields(jsonStr)
	return jsonStr
}

// CleanJSONSchemaForGemini transforms a JSON schema to be compatible with Gemini tool calling.
// It removes unsupported keywords and simplifies schemas, without adding empty-schema placeholders.
func CleanJSONSchemaForGemini(jsonStr string) string {
	return cleanJSONSchema(jsonStr, false)
}

// cleanJSONSchema performs the core cleaning operations on the JSON schema.
func cleanJSONSchema(jsonStr string, addPlaceholder bool) string {
	// Phase 0: Normalize malformed schemas (e.g. bare property maps and boolean required from MCP tools)
	jsonStr = normalizeMalformedSchemaObjects(jsonStr, true)

	// Phase 1: Convert and add hints
	jsonStr = convertRefsToHints(jsonStr)
	jsonStr = convertConstToEnum(jsonStr)
	jsonStr = convertEnumValuesToStrings(jsonStr)
	jsonStr = addEnumHints(jsonStr)
	jsonStr = addAdditionalPropertiesHints(jsonStr)
	jsonStr = moveConstraintsToDescription(jsonStr)

	// Phase 2: Flatten complex structures
	jsonStr = mergeAllOf(jsonStr)
	jsonStr = flattenAnyOfOneOf(jsonStr)
	jsonStr = flattenTypeArrays(jsonStr)

	// Phase 3: Cleanup
	jsonStr = removeUnsupportedKeywords(jsonStr)
	// Remove schema metadata title values while preserving properties named "title".
	jsonStr = removeKeywords(jsonStr, []string{"title"})
	if !addPlaceholder {
		// Gemini schema cleanup: remove nullable and placeholder-only fields.
		jsonStr = removeKeywords(jsonStr, []string{"nullable"})
		jsonStr = removePlaceholderFields(jsonStr)
	}
	jsonStr = cleanupRequiredFields(jsonStr)
	// Phase 4: Add placeholder for empty object schemas (Claude VALIDATED mode requirement)
	if addPlaceholder {
		jsonStr = addEmptySchemaPlaceholder(jsonStr)
	}

	return jsonStr
}

// removeKeywords removes all occurrences of specified keywords from the JSON schema.
func removeKeywords(jsonStr string, keywords []string) string {
	deletePaths := make([]string, 0)
	pathsByField := findPathsByFields(jsonStr, keywords)
	for _, key := range keywords {
		for _, p := range pathsByField[key] {
			if isPropertyDefinition(trimSuffix(p, "."+key)) {
				continue
			}
			deletePaths = append(deletePaths, p)
		}
	}
	sortByDepth(deletePaths)
	for _, p := range deletePaths {
		jsonStr, _ = sjson.Delete(jsonStr, p)
	}
	return jsonStr
}

// removePlaceholderFields removes placeholder-only properties ("_" and "reason") and their required entries.
func removePlaceholderFields(jsonStr string) string {
	// Remove "_" placeholder properties.
	paths := findPaths(jsonStr, "_")
	sortByDepth(paths)
	for _, p := range paths {
		if !strings.HasSuffix(p, ".properties._") {
			continue
		}
		jsonStr, _ = sjson.Delete(jsonStr, p)
		parentPath := trimSuffix(p, ".properties._")
		reqPath := joinPath(parentPath, "required")
		req := gjson.Get(jsonStr, reqPath)
		if req.IsArray() {
			var filtered []string
			for _, r := range req.Array() {
				if r.String() != "_" {
					filtered = append(filtered, r.String())
				}
			}
			if len(filtered) == 0 {
				jsonStr, _ = sjson.Delete(jsonStr, reqPath)
			} else {
				updated, _ := sjson.SetBytes([]byte(jsonStr), reqPath, filtered)
				jsonStr = string(updated)
			}
		}
	}

	// Remove placeholder-only "reason" objects.
	reasonPaths := findPaths(jsonStr, "reason")
	sortByDepth(reasonPaths)
	for _, p := range reasonPaths {
		if !strings.HasSuffix(p, ".properties.reason") {
			continue
		}
		parentPath := trimSuffix(p, ".properties.reason")
		props := gjson.Get(jsonStr, joinPath(parentPath, "properties"))
		if !props.IsObject() || len(props.Map()) != 1 {
			continue
		}
		desc := gjson.Get(jsonStr, p+".description").String()
		if desc != placeholderReasonDescription {
			continue
		}
		jsonStr, _ = sjson.Delete(jsonStr, p)
		reqPath := joinPath(parentPath, "required")
		req := gjson.Get(jsonStr, reqPath)
		if req.IsArray() {
			var filtered []string
			for _, r := range req.Array() {
				if r.String() != "reason" {
					filtered = append(filtered, r.String())
				}
			}
			if len(filtered) == 0 {
				jsonStr, _ = sjson.Delete(jsonStr, reqPath)
			} else {
				updated, _ := sjson.SetBytes([]byte(jsonStr), reqPath, filtered)
				jsonStr = string(updated)
			}
		}
	}

	return jsonStr
}

// normalizeMalformedSchemaObjects normalizes malformed JSON schema nodes commonly produced by
// certain MCP tool definitions (e.g. Asana MCP server):
// 1. Bare property maps missing the "type": "object" and "properties": {...} wrappers are wrapped.
// 2. Boolean "required": true on property definitions are stripped and promoted to the parent's "required" array.
// 3. Tool array schemas missing "items" receive a string item schema required by Gemini and Antigravity.
func normalizeMalformedSchemaObjects(jsonStr string, addMissingArrayItems bool) string {
	if jsonStr == "" {
		return jsonStr
	}

	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return jsonStr
	}

	rootMap, ok := root.(map[string]any)
	if !ok || isAPIRequestDocument(rootMap) {
		return jsonStr
	}

	// If wrapped in single-key {"schema": ...} by cleanNestedSchema, unwrap, repair, and re-wrap.
	if len(rootMap) == 1 {
		if innerSchema, ok := rootMap["schema"].(map[string]any); ok {
			repairedInner, modified := repairSchemaNode(innerSchema, addMissingArrayItems)
			if !modified {
				return jsonStr
			}
			out, err := marshalJSONNoHTMLEscape(map[string]any{"schema": repairedInner})
			if err != nil {
				return jsonStr
			}
			return string(out)
		}
	}

	repaired, modified := repairSchemaNode(rootMap, addMissingArrayItems)
	if !modified {
		return jsonStr
	}

	out, err := marshalJSONNoHTMLEscape(repaired)
	if err != nil {
		return jsonStr
	}
	return string(out)
}

func marshalJSONNoHTMLEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

func isKnownSchemaKeywordOrExtension(key string) bool {
	if strings.HasPrefix(key, "x-") {
		return true
	}
	switch key {
	case "properties", "patternProperties", "additionalProperties", "items", "prefixItems",
		"$defs", "definitions", "dependentSchemas", "dependentRequired", "dependencies",
		"if", "then", "else", "not", "contains", "propertyNames",
		"unevaluatedProperties", "unevaluatedItems", "contentSchema", "additionalItems",
		"default", "const", "example", "examples", "discriminator", "xml", "externalDocs",
		"enumDescriptions", "enumTitles":
		return true
	}
	return false
}

func isNonObjectDeclaredType(t any) bool {
	if s, ok := t.(string); ok {
		return s != "" && s != "object"
	}
	if arr, ok := t.([]any); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok && s == "object" {
				return false
			}
		}
		return len(arr) > 0
	}
	return false
}

func isArrayDeclaredType(t any) bool {
	switch typeValue := t.(type) {
	case string:
		return typeValue == "array"
	case []any:
		for _, item := range typeValue {
			if itemType, ok := item.(string); ok && itemType == "array" {
				return true
			}
		}
	}
	return false
}

func isAPIRequestDocument(m map[string]any) bool {
	if _, ok := m["tools"].([]any); ok {
		return true
	}
	if _, ok := m["contents"].([]any); ok {
		return true
	}
	if _, ok := m["messages"].([]any); ok {
		return true
	}
	if _, ok := m["functionDeclarations"].([]any); ok {
		return true
	}
	if _, ok := m["function_declarations"].([]any); ok {
		return true
	}
	if reqMap, ok := m["request"].(map[string]any); ok {
		if isAPIRequestDocument(reqMap) {
			return true
		}
	}
	return false
}

func repairSchemaNode(node map[string]any, addMissingArrayItems bool) (map[string]any, bool) {
	if node == nil {
		return nil, false
	}

	modified := false
	clone := make(map[string]any, len(node))
	for k, v := range node {
		clone[k] = v
	}

	// 1. If not declared as a primitive/array type, collect bare property definition maps
	if !isNonObjectDeclaredType(clone["type"]) {
		var bareProps map[string]any
		for k, v := range clone {
			if childMap, isMap := v.(map[string]any); isMap {
				if !isKnownSchemaKeywordOrExtension(k) {
					if bareProps == nil {
						bareProps = make(map[string]any)
					}
					bareProps[k] = childMap
				}
			}
		}

		if len(bareProps) > 0 {
			repairedProps, promotedReqs, _ := repairPropertyMap(bareProps, addMissingArrayItems)
			for k := range bareProps {
				delete(clone, k)
			}

			if existingProps, ok := clone["properties"].(map[string]any); ok {
				newProps := make(map[string]any, len(existingProps)+len(repairedProps))
				for k, v := range existingProps {
					newProps[k] = v
				}
				for k, v := range repairedProps {
					newProps[k] = v
				}
				clone["properties"] = newProps
			} else {
				clone["properties"] = repairedProps
				if _, hasType := clone["type"]; !hasType {
					clone["type"] = "object"
				}
			}

			if len(promotedReqs) > 0 {
				existingReqs := extractStringArray(clone["required"])
				merged := mergeStringSlices(existingReqs, promotedReqs)
				clone["required"] = merged
			}
			modified = true
		}
	}

	// 2. If node has a "properties" map, recursively repair all properties inside it
	if propsVal, ok := clone["properties"].(map[string]any); ok {
		repairedProps, promotedReqs, propsMod := repairPropertyMap(propsVal, addMissingArrayItems)
		if propsMod {
			clone["properties"] = repairedProps
			modified = true
		}
		if len(promotedReqs) > 0 {
			existingReqs := extractStringArray(clone["required"])
			merged := mergeStringSlices(existingReqs, promotedReqs)
			clone["required"] = merged
			modified = true
		}
	}

	// Gemini and Antigravity reject tool array schemas without an items definition.
	if addMissingArrayItems && isArrayDeclaredType(clone["type"]) {
		if _, hasItems := clone["items"]; !hasItems {
			clone["items"] = map[string]any{"type": "string"}
			modified = true
		}
	}

	// 3. Recurse into all other standard schema containers
	if itemsVal, ok := clone["items"].(map[string]any); ok {
		repairedItems, itemsMod := repairSchemaNode(itemsVal, addMissingArrayItems)
		if itemsMod {
			clone["items"] = repairedItems
			modified = true
		}
	} else if itemsList, ok := clone["items"].([]any); ok {
		repairedList, listMod := repairSchemaList(itemsList, addMissingArrayItems)
		if listMod {
			clone["items"] = repairedList
			modified = true
		}
	}

	if addProps, ok := clone["additionalProperties"].(map[string]any); ok {
		repairedAddProps, addPropsMod := repairSchemaNode(addProps, addMissingArrayItems)
		if addPropsMod {
			clone["additionalProperties"] = repairedAddProps
			modified = true
		}
	}

	if patProps, ok := clone["patternProperties"].(map[string]any); ok {
		repairedPatProps, _, patMod := repairPropertyMap(patProps, addMissingArrayItems)
		if patMod {
			clone["patternProperties"] = repairedPatProps
			modified = true
		}
	}

	for _, key := range []string{"if", "then", "else", "not", "contains", "propertyNames", "unevaluatedProperties", "unevaluatedItems", "contentSchema", "additionalItems"} {
		if subVal, ok := clone[key].(map[string]any); ok {
			repairedSub, subMod := repairSchemaNode(subVal, addMissingArrayItems)
			if subMod {
				clone[key] = repairedSub
				modified = true
			}
		}
	}

	for _, key := range []string{"anyOf", "oneOf", "allOf", "prefixItems"} {
		if listVal, ok := clone[key].([]any); ok {
			repairedList, listMod := repairSchemaList(listVal, addMissingArrayItems)
			if listMod {
				clone[key] = repairedList
				modified = true
			}
		}
	}

	for _, key := range []string{"$defs", "definitions", "dependentSchemas"} {
		if defsVal, ok := clone[key].(map[string]any); ok {
			repairedDefs := make(map[string]any, len(defsVal))
			defsModified := false
			for dk, dv := range defsVal {
				if defMap, ok := dv.(map[string]any); ok {
					repairedDef, defMod := repairSchemaNode(defMap, addMissingArrayItems)
					repairedDefs[dk] = repairedDef
					if defMod {
						defsModified = true
						modified = true
					}
				} else {
					repairedDefs[dk] = dv
				}
			}
			if defsModified {
				clone[key] = repairedDefs
			}
		}
	}

	return clone, modified
}

func repairSchemaList(list []any, addMissingArrayItems bool) ([]any, bool) {
	var repairedList []any
	listModified := false
	for _, item := range list {
		if itemMap, ok := item.(map[string]any); ok {
			repairedItem, itemMod := repairSchemaNode(itemMap, addMissingArrayItems)
			repairedList = append(repairedList, repairedItem)
			if itemMod {
				listModified = true
			}
		} else {
			repairedList = append(repairedList, item)
		}
	}
	return repairedList, listModified
}

func repairPropertyMap(props map[string]any, addMissingArrayItems bool) (map[string]any, []string, bool) {
	out := make(map[string]any, len(props))
	var promotedReqs []string
	modified := false

	for k, v := range props {
		childMap, isMap := v.(map[string]any)
		if !isMap {
			out[k] = v
			continue
		}

		childClone := make(map[string]any, len(childMap))
		for ck, cv := range childMap {
			childClone[ck] = cv
		}

		if reqBool, isBool := childClone["required"].(bool); isBool {
			delete(childClone, "required")
			modified = true
			if reqBool {
				promotedReqs = append(promotedReqs, k)
			}
		}

		repairedChild, childMod := repairSchemaNode(childClone, addMissingArrayItems)
		if childMod {
			modified = true
		}
		out[k] = repairedChild
	}

	sort.Strings(promotedReqs)
	return out, promotedReqs, modified
}

func extractStringArray(val any) []string {
	if val == nil {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		if strArr, ok := val.([]string); ok {
			return strArr
		}
		return nil
	}
	var res []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			res = append(res, s)
		}
	}
	return res
}

func mergeStringSlices(existing, promoted []string) []string {
	seen := make(map[string]bool)
	var res []string
	for _, s := range existing {
		if !seen[s] && s != "" {
			seen[s] = true
			res = append(res, s)
		}
	}
	for _, s := range promoted {
		if !seen[s] && s != "" {
			seen[s] = true
			res = append(res, s)
		}
	}
	return res
}

// convertRefsToHints converts $ref to description hints (Lazy Hint strategy).
func convertRefsToHints(jsonStr string) string {
	paths := findPaths(jsonStr, "$ref")
	sortByDepth(paths)

	for _, p := range paths {
		refVal := gjson.Get(jsonStr, p).String()
		defName := refVal
		if idx := strings.LastIndex(refVal, "/"); idx >= 0 {
			defName = refVal[idx+1:]
		}

		parentPath := trimSuffix(p, ".$ref")
		hint := fmt.Sprintf("See: %s", defName)
		if existing := gjson.Get(jsonStr, descriptionPath(parentPath)).String(); existing != "" {
			hint = fmt.Sprintf("%s (%s)", existing, hint)
		}

		replacement := `{"type":"object","description":""}`
		replacementBytes, _ := sjson.SetBytes([]byte(replacement), "description", hint)
		replacement = string(replacementBytes)
		jsonStr = setRawAt(jsonStr, parentPath, replacement)
	}
	return jsonStr
}

func convertConstToEnum(jsonStr string) string {
	for _, p := range findPaths(jsonStr, "const") {
		val := gjson.Get(jsonStr, p)
		if !val.Exists() {
			continue
		}
		enumPath := trimSuffix(p, ".const") + ".enum"
		if !gjson.Get(jsonStr, enumPath).Exists() {
			updated, _ := sjson.SetBytes([]byte(jsonStr), enumPath, []interface{}{val.Value()})
			jsonStr = string(updated)
		}
	}
	return jsonStr
}

// convertEnumValuesToStrings ensures all enum values are strings and the schema type is set to string.
// Gemini API requires enum values to be of type string, not numbers or booleans.
func convertEnumValuesToStrings(jsonStr string) string {
	for _, p := range findPaths(jsonStr, "enum") {
		arr := gjson.Get(jsonStr, p)
		if !arr.IsArray() {
			continue
		}

		var stringVals []string
		for _, item := range arr.Array() {
			stringVals = append(stringVals, item.String())
		}

		// Always update enum values to strings and set type to "string"
		// This ensures compatibility with Antigravity Gemini which only allows enum for STRING type
		updated, _ := sjson.SetBytes([]byte(jsonStr), p, stringVals)
		jsonStr = string(updated)
		parentPath := trimSuffix(p, ".enum")
		updated, _ = sjson.SetBytes([]byte(jsonStr), joinPath(parentPath, "type"), "string")
		jsonStr = string(updated)
	}
	return jsonStr
}

func convertEnumValuesToStringsPreservingType(jsonStr string) string {
	for _, p := range findPaths(jsonStr, "enum") {
		arr := gjson.Get(jsonStr, p)
		if !arr.IsArray() {
			continue
		}

		var stringVals []string
		for _, item := range arr.Array() {
			stringVals = append(stringVals, item.String())
		}

		updated, _ := sjson.SetBytes([]byte(jsonStr), p, stringVals)
		jsonStr = string(updated)
	}
	return jsonStr
}

func addEnumHints(jsonStr string) string {
	for _, p := range findPaths(jsonStr, "enum") {
		arr := gjson.Get(jsonStr, p)
		if !arr.IsArray() {
			continue
		}
		items := arr.Array()
		if len(items) <= 1 || len(items) > 10 {
			continue
		}

		var vals []string
		for _, item := range items {
			vals = append(vals, item.String())
		}
		jsonStr = appendHint(jsonStr, trimSuffix(p, ".enum"), "Allowed: "+strings.Join(vals, ", "))
	}
	return jsonStr
}

func addAdditionalPropertiesHints(jsonStr string) string {
	for _, p := range findPaths(jsonStr, "additionalProperties") {
		if gjson.Get(jsonStr, p).Type == gjson.False {
			jsonStr = appendHint(jsonStr, trimSuffix(p, ".additionalProperties"), "No extra properties allowed")
		}
	}
	return jsonStr
}

var unsupportedConstraints = []string{
	"minLength", "maxLength", "exclusiveMinimum", "exclusiveMaximum",
	"pattern", "minItems", "maxItems", "uniqueItems", "contains", "format",
	"default", "examples", // Claude rejects these in VALIDATED mode
}

func moveConstraintsToDescription(jsonStr string) string {
	pathsByField := findPathsByFields(jsonStr, unsupportedConstraints)
	for _, key := range unsupportedConstraints {
		for _, p := range pathsByField[key] {
			val := gjson.Get(jsonStr, p)
			if !val.Exists() {
				continue
			}
			parentPath := trimSuffix(p, "."+key)
			if isPropertyDefinition(parentPath) {
				continue
			}
			if val.IsObject() || val.IsArray() {
				jsonStr = appendHint(jsonStr, parentPath, fmt.Sprintf("%s: %s", key, val.Raw))
				continue
			}
			jsonStr = appendHint(jsonStr, parentPath, fmt.Sprintf("%s: %s", key, val.String()))
		}
	}
	return jsonStr
}

func mergeAllOf(jsonStr string) string {
	paths := findPaths(jsonStr, "allOf")
	sortByDepth(paths)

	for _, p := range paths {
		allOf := gjson.Get(jsonStr, p)
		if !allOf.IsArray() {
			continue
		}
		parentPath := trimSuffix(p, ".allOf")

		for _, item := range allOf.Array() {
			if props := item.Get("properties"); props.IsObject() {
				props.ForEach(func(key, value gjson.Result) bool {
					destPath := joinPath(parentPath, "properties."+escapeGJSONPathKey(key.String()))
					updated, _ := sjson.SetRawBytes([]byte(jsonStr), destPath, []byte(value.Raw))
					jsonStr = string(updated)
					return true
				})
			}
			if req := item.Get("required"); req.IsArray() {
				reqPath := joinPath(parentPath, "required")
				current := getStrings(jsonStr, reqPath)
				for _, r := range req.Array() {
					if s := r.String(); !contains(current, s) {
						current = append(current, s)
					}
				}
				updated, _ := sjson.SetBytes([]byte(jsonStr), reqPath, current)
				jsonStr = string(updated)
			}
		}
		jsonStr, _ = sjson.Delete(jsonStr, p)
	}
	return jsonStr
}

func mergeMissingSchemaAtPath(jsonStr, destination string, incoming gjson.Result) string {
	existing := gjson.Get(jsonStr, destination)
	if !existing.Exists() {
		updated, _ := sjson.SetRawBytes([]byte(jsonStr), destination, []byte(incoming.Raw))
		return string(updated)
	}
	if !existing.IsObject() || !incoming.IsObject() {
		return jsonStr
	}
	incoming.ForEach(func(key, value gjson.Result) bool {
		child := joinPath(destination, escapeGJSONPathKey(key.String()))
		jsonStr = mergeMissingSchemaAtPath(jsonStr, child, value)
		return true
	})
	return jsonStr
}

func flattenAnyOfOneOf(jsonStr string) string {
	for _, key := range []string{"anyOf", "oneOf"} {
		paths := findPaths(jsonStr, key)
		sortByDepth(paths)

		for _, p := range paths {
			arr := gjson.Get(jsonStr, p)
			if !arr.IsArray() || len(arr.Array()) == 0 {
				continue
			}

			parentPath := trimSuffix(p, "."+key)
			parent := gjson.Get(jsonStr, parentPath)
			if parentPath == "" {
				parent = gjson.Parse(jsonStr)
			}

			items := arr.Array()

			if parentProps := parent.Get("properties"); parentProps.IsObject() {
				hasNull := false
				for _, item := range items {
					if item.Get("type").String() == "null" {
						hasNull = true
					}
					if branchProps := item.Get("properties"); branchProps.IsObject() {
						branchProps.ForEach(func(propKey, propVal gjson.Result) bool {
							destPath := joinPath(parentPath, "properties."+escapeGJSONPathKey(propKey.String()))
							jsonStr = mergeMissingSchemaAtPath(jsonStr, destPath, propVal)
							return true
						})
					}
				}
				if hasNull {
					updated, _ := sjson.SetBytes([]byte(jsonStr), joinPath(parentPath, "nullable"), true)
					jsonStr = string(updated)
				}
				jsonStr, _ = sjson.Delete(jsonStr, p)
				continue
			}

			parentDesc := gjson.Get(jsonStr, descriptionPath(parentPath)).String()
			bestIdx, allTypes := selectBest(items)
			selected := items[bestIdx].Raw

			if parentDesc != "" {
				selected = mergeDescriptionRaw(selected, parentDesc)
			}

			if len(allTypes) > 1 {
				hint := "Accepts: " + strings.Join(allTypes, " | ")
				selected = appendHintRaw(selected, hint)
			}

			jsonStr = setRawAt(jsonStr, parentPath, selected)
		}
	}
	return jsonStr
}

func selectBest(items []gjson.Result) (bestIdx int, types []string) {
	bestScore := -1
	for i, item := range items {
		t := item.Get("type").String()
		score := 0

		switch {
		case t == "object" || item.Get("properties").Exists():
			score, t = 3, orDefault(t, "object")
		case t == "array" || item.Get("items").Exists():
			score, t = 2, orDefault(t, "array")
		case t != "" && t != "null":
			score = 1
		case t == "null":
			score, t = 0, "null"
		default:
			score, t = 0, ""
		}

		if t != "" {
			types = append(types, t)
		}
		if score > bestScore {
			bestScore, bestIdx = score, i
		}
	}
	return
}

func flattenTypeArrays(jsonStr string) string {
	paths := findPaths(jsonStr, "type")
	sortByDepth(paths)

	nullableFields := make(map[string][]string)

	for _, p := range paths {
		res := gjson.Get(jsonStr, p)
		if !res.IsArray() || len(res.Array()) == 0 {
			continue
		}

		hasNull := false
		var nonNullTypes []string
		for _, item := range res.Array() {
			s := item.String()
			if s == "null" {
				hasNull = true
			} else if s != "" {
				nonNullTypes = append(nonNullTypes, s)
			}
		}

		firstType := "string"
		if len(nonNullTypes) > 0 {
			firstType = nonNullTypes[0]
		}

		updated, _ := sjson.SetBytes([]byte(jsonStr), p, firstType)
		jsonStr = string(updated)

		parentPath := trimSuffix(p, ".type")
		if len(nonNullTypes) > 1 {
			hint := "Accepts: " + strings.Join(nonNullTypes, " | ")
			jsonStr = appendHint(jsonStr, parentPath, hint)
		}

		if hasNull {
			parts := splitGJSONPath(p)
			if len(parts) >= 3 && parts[len(parts)-3] == "properties" {
				fieldNameEscaped := parts[len(parts)-2]
				fieldName := unescapeGJSONPathKey(fieldNameEscaped)
				objectPath := strings.Join(parts[:len(parts)-3], ".")
				nullableFields[objectPath] = append(nullableFields[objectPath], fieldName)

				propPath := joinPath(objectPath, "properties."+fieldNameEscaped)
				jsonStr = appendHint(jsonStr, propPath, "(nullable)")
			}
		}
	}

	for objectPath, fields := range nullableFields {
		reqPath := joinPath(objectPath, "required")
		req := gjson.Get(jsonStr, reqPath)
		if !req.IsArray() {
			continue
		}

		var filtered []string
		for _, r := range req.Array() {
			if !contains(fields, r.String()) {
				filtered = append(filtered, r.String())
			}
		}

		if len(filtered) == 0 {
			jsonStr, _ = sjson.Delete(jsonStr, reqPath)
		} else {
			updated, _ := sjson.SetBytes([]byte(jsonStr), reqPath, filtered)
			jsonStr = string(updated)
		}
	}
	return jsonStr
}

func removeUnsupportedKeywords(jsonStr string) string {
	keywords := append(unsupportedConstraints,
		"$schema", "$defs", "definitions", "const", "$ref", "$id", "$comment", "additionalProperties",
		"propertyNames", "patternProperties", // Gemini doesn't support these schema keywords
		"enumTitles", "prefill", "deprecated", "encrypted", // Schema metadata fields unsupported by Gemini
	)

	deletePaths := make([]string, 0)
	pathsByField := findPathsByFields(jsonStr, keywords)
	for _, key := range keywords {
		for _, p := range pathsByField[key] {
			if isPropertyDefinition(trimSuffix(p, "."+key)) {
				continue
			}
			deletePaths = append(deletePaths, p)
		}
	}
	sortByDepth(deletePaths)
	for _, p := range deletePaths {
		jsonStr, _ = sjson.Delete(jsonStr, p)
	}
	// Remove x-* extension fields (e.g., x-google-enum-descriptions) that are not supported by Gemini API
	jsonStr = removeExtensionFields(jsonStr)
	return jsonStr
}

// removeExtensionFields removes all x-* extension fields from the JSON schema.
// These are OpenAPI/JSON Schema extension fields that Google APIs don't recognize.
func removeExtensionFields(jsonStr string) string {
	var paths []string
	walkForExtensions(gjson.Parse(jsonStr), "", &paths)
	// walkForExtensions returns paths in a way that deeper paths are added before their ancestors
	// when they are not deleted wholesale, but since we skip children of deleted x-* nodes,
	// any collected path is safe to delete. We still use DeleteBytes for efficiency.

	b := []byte(jsonStr)
	for _, p := range paths {
		b, _ = sjson.DeleteBytes(b, p)
	}
	return string(b)
}

func walkForExtensions(value gjson.Result, path string, paths *[]string) {
	if value.IsArray() {
		arr := value.Array()
		for i := len(arr) - 1; i >= 0; i-- {
			walkForExtensions(arr[i], joinPath(path, strconv.Itoa(i)), paths)
		}
		return
	}

	if value.IsObject() {
		value.ForEach(func(key, val gjson.Result) bool {
			keyStr := key.String()
			safeKey := escapeGJSONPathKey(keyStr)
			childPath := joinPath(path, safeKey)

			// If it's an extension field, we delete it and don't need to look at its children.
			if strings.HasPrefix(keyStr, "x-") && !isPropertyDefinition(path) {
				*paths = append(*paths, childPath)
				return true
			}

			walkForExtensions(val, childPath, paths)
			return true
		})
	}
}

func cleanupRequiredFields(jsonStr string) string {
	for _, p := range findPaths(jsonStr, "required") {
		parentPath := trimSuffix(p, ".required")
		propsPath := joinPath(parentPath, "properties")

		req := gjson.Get(jsonStr, p)
		props := gjson.Get(jsonStr, propsPath)
		if !req.IsArray() {
			continue
		}
		if !props.IsObject() {
			jsonStr, _ = sjson.Delete(jsonStr, p)
			continue
		}

		var valid []string
		for _, r := range req.Array() {
			key := r.String()
			if props.Get(escapeGJSONPathKey(key)).Exists() {
				valid = append(valid, key)
			}
		}

		if len(valid) != len(req.Array()) {
			if len(valid) == 0 {
				jsonStr, _ = sjson.Delete(jsonStr, p)
			} else {
				updated, _ := sjson.SetBytes([]byte(jsonStr), p, valid)
				jsonStr = string(updated)
			}
		}
	}
	return jsonStr
}

// addEmptySchemaPlaceholder adds a placeholder "reason" property to empty object schemas.
// Claude VALIDATED mode requires at least one required property in tool schemas.
func addEmptySchemaPlaceholder(jsonStr string) string {
	// Find all "type" fields
	paths := findPaths(jsonStr, "type")

	// Process from deepest to shallowest (to handle nested objects properly)
	sortByDepth(paths)

	for _, p := range paths {
		typeVal := gjson.Get(jsonStr, p)
		if typeVal.String() != "object" {
			continue
		}

		// Get the parent path (the object containing "type")
		parentPath := trimSuffix(p, ".type")

		// Check if properties exists and is empty or missing
		propsPath := joinPath(parentPath, "properties")
		propsVal := gjson.Get(jsonStr, propsPath)
		reqPath := joinPath(parentPath, "required")
		reqVal := gjson.Get(jsonStr, reqPath)
		hasRequiredProperties := reqVal.IsArray() && len(reqVal.Array()) > 0

		needsPlaceholder := false
		if !propsVal.Exists() {
			// No properties field at all
			needsPlaceholder = true
		} else if propsVal.IsObject() && len(propsVal.Map()) == 0 {
			// Empty properties object
			needsPlaceholder = true
		}

		if needsPlaceholder {
			// Add placeholder "reason" property
			reasonPath := joinPath(propsPath, "reason")
			updated, _ := sjson.SetBytes([]byte(jsonStr), reasonPath+".type", "string")
			jsonStr = string(updated)
			updated, _ = sjson.SetBytes([]byte(jsonStr), reasonPath+".description", placeholderReasonDescription)
			jsonStr = string(updated)

			// Add to required array
			updated, _ = sjson.SetBytes([]byte(jsonStr), reqPath, []string{"reason"})
			jsonStr = string(updated)
			continue
		}

		// If schema has properties but none are required, add a minimal placeholder.
		if propsVal.IsObject() && !hasRequiredProperties {
			// DO NOT add placeholder if it's a top-level schema (parentPath is empty)
			// or if we've already added a placeholder reason above.
			if parentPath == "" {
				continue
			}
			placeholderPath := joinPath(propsPath, "_")
			if !gjson.Get(jsonStr, placeholderPath).Exists() {
				updated, _ := sjson.SetBytes([]byte(jsonStr), placeholderPath+".type", "boolean")
				jsonStr = string(updated)
			}
			updated, _ := sjson.SetBytes([]byte(jsonStr), reqPath, []string{"_"})
			jsonStr = string(updated)
		}
	}

	return jsonStr
}

// --- Helpers ---

func findPaths(jsonStr, field string) []string {
	var paths []string
	Walk(gjson.Parse(jsonStr), "", field, &paths)
	return paths
}

func findPathsByFields(jsonStr string, fields []string) map[string][]string {
	set := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		set[field] = struct{}{}
	}
	paths := make(map[string][]string, len(set))
	walkForFields(gjson.Parse(jsonStr), "", set, paths)
	return paths
}

func walkForFields(value gjson.Result, path string, fields map[string]struct{}, paths map[string][]string) {
	switch value.Type {
	case gjson.JSON:
		value.ForEach(func(key, val gjson.Result) bool {
			keyStr := key.String()
			safeKey := escapeGJSONPathKey(keyStr)

			var childPath string
			if path == "" {
				childPath = safeKey
			} else {
				childPath = path + "." + safeKey
			}

			if _, ok := fields[keyStr]; ok {
				paths[keyStr] = append(paths[keyStr], childPath)
			}

			walkForFields(val, childPath, fields, paths)
			return true
		})
	case gjson.String, gjson.Number, gjson.True, gjson.False, gjson.Null:
		// Terminal types - no further traversal needed
	}
}

func sortByDepth(paths []string) {
	sort.Slice(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
}

func trimSuffix(path, suffix string) string {
	if path == strings.TrimPrefix(suffix, ".") {
		return ""
	}
	return strings.TrimSuffix(path, suffix)
}

func joinPath(base, suffix string) string {
	if base == "" {
		return suffix
	}
	return base + "." + suffix
}

func setRawAt(jsonStr, path, value string) string {
	if path == "" {
		return value
	}
	result, _ := sjson.SetRawBytes([]byte(jsonStr), path, []byte(value))
	return string(result)
}

func isPropertyDefinition(path string) bool {
	return path == "properties" || strings.HasSuffix(path, ".properties")
}

func descriptionPath(parentPath string) string {
	if parentPath == "" || parentPath == "@this" {
		return "description"
	}
	return parentPath + ".description"
}

func appendHint(jsonStr, parentPath, hint string) string {
	descPath := parentPath + ".description"
	if parentPath == "" || parentPath == "@this" {
		descPath = "description"
	}
	existing := gjson.Get(jsonStr, descPath).String()
	if existing != "" {
		hint = fmt.Sprintf("%s (%s)", existing, hint)
	}
	updated, _ := sjson.SetBytes([]byte(jsonStr), descPath, hint)
	jsonStr = string(updated)
	return jsonStr
}

func appendHintRaw(jsonRaw, hint string) string {
	existing := gjson.Get(jsonRaw, "description").String()
	if existing != "" {
		hint = fmt.Sprintf("%s (%s)", existing, hint)
	}
	updated, _ := sjson.SetBytes([]byte(jsonRaw), "description", hint)
	jsonRaw = string(updated)
	return jsonRaw
}

func getStrings(jsonStr, path string) []string {
	var result []string
	if arr := gjson.Get(jsonStr, path); arr.IsArray() {
		for _, r := range arr.Array() {
			result = append(result, r.String())
		}
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func orDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func escapeGJSONPathKey(key string) string {
	if strings.IndexAny(key, ".*?") == -1 {
		return key
	}
	return gjsonPathKeyReplacer.Replace(key)
}

func unescapeGJSONPathKey(key string) string {
	if !strings.Contains(key, "\\") {
		return key
	}
	var b strings.Builder
	b.Grow(len(key))
	for i := 0; i < len(key); i++ {
		if key[i] == '\\' && i+1 < len(key) {
			i++
			b.WriteByte(key[i])
			continue
		}
		b.WriteByte(key[i])
	}
	return b.String()
}

func splitGJSONPath(path string) []string {
	if path == "" {
		return nil
	}

	parts := make([]string, 0, strings.Count(path, ".")+1)
	var b strings.Builder
	b.Grow(len(path))

	for i := 0; i < len(path); i++ {
		c := path[i]
		if c == '\\' && i+1 < len(path) {
			b.WriteByte('\\')
			i++
			b.WriteByte(path[i])
			continue
		}
		if c == '.' {
			parts = append(parts, b.String())
			b.Reset()
			continue
		}
		b.WriteByte(c)
	}
	parts = append(parts, b.String())
	return parts
}

func mergeDescriptionRaw(schemaRaw, parentDesc string) string {
	childDesc := gjson.Get(schemaRaw, "description").String()
	switch {
	case childDesc == "":
		updated, _ := sjson.SetBytes([]byte(schemaRaw), "description", parentDesc)
		return string(updated)
	case childDesc == parentDesc:
		return schemaRaw
	default:
		combined := fmt.Sprintf("%s (%s)", parentDesc, childDesc)
		updated, _ := sjson.SetBytes([]byte(schemaRaw), "description", combined)
		return string(updated)
	}
}

// InlineLocalRefs resolves JSON Pointer references against the original schema before definition
// containers are stripped. Each expansion receives its own copy, sibling keywords override the
// referenced definition, and cycles terminate as a typed hint instead of recursing forever.
func InlineLocalRefs(jsonStr string) string {
	return inlineLocalRefs(jsonStr)
}

func inlineLocalRefs(jsonStr string) string {
	if !strings.Contains(jsonStr, `"$ref"`) {
		return jsonStr
	}

	decoder := json.NewDecoder(strings.NewReader(jsonStr))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return jsonStr
	}

	resolved := resolveLocalRefs(root, root, make(map[string]bool))
	out, err := json.Marshal(resolved)
	if err != nil {
		return jsonStr
	}
	return string(out)
}

func resolveLocalRefs(root, value any, active map[string]bool) any {
	switch node := value.(type) {
	case []any:
		out := make([]any, len(node))
		for i, item := range node {
			out[i] = resolveLocalRefs(root, item, active)
		}
		return out
	case map[string]any:
		ref, hasRef := node["$ref"].(string)
		if hasRef && strings.HasPrefix(ref, "#/") {
			if target, ok := resolveJSONPointer(root, ref); ok {
				if active[ref] {
					return cyclicRefFallback(node, target, ref)
				}
				active[ref] = true
				resolvedTarget := resolveLocalRefs(root, target, active)
				delete(active, ref)
				if targetMap, okTarget := resolvedTarget.(map[string]any); okTarget {
					out := make(map[string]any, len(targetMap)+len(node))
					for key, item := range targetMap {
						out[key] = item
					}
					for key, item := range node {
						if key == "$ref" {
							continue
						}
						out[key] = resolveLocalRefs(root, item, active)
					}
					return out
				}
			}
		}

		out := make(map[string]any, len(node))
		for key, item := range node {
			out[key] = resolveLocalRefs(root, item, active)
		}
		return out
	default:
		return value
	}
}

func resolveJSONPointer(root any, ref string) (any, bool) {
	current := root
	for _, rawPart := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		part := strings.ReplaceAll(strings.ReplaceAll(rawPart, "~1", "/"), "~0", "~")
		switch node := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = node[part]
			if !ok {
				return nil, false
			}
		case []any:
			index, err := strconv.Atoi(part)
			if err != nil || index < 0 || index >= len(node) {
				return nil, false
			}
			current = node[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func cyclicRefFallback(node map[string]any, target any, ref string) map[string]any {
	out := make(map[string]any, len(node)+2)
	if targetMap, ok := target.(map[string]any); ok {
		for _, key := range []string{"type", "nullable", "description"} {
			if value, exists := targetMap[key]; exists {
				out[key] = value
			}
		}
	}
	for key, value := range node {
		if key != "$ref" {
			out[key] = value
		}
	}
	name := refName(ref)
	hint := "See: " + name
	if description, _ := out["description"].(string); description != "" {
		out["description"] = mergeHint(description, hint)
	} else {
		out["description"] = hint
	}
	return out
}

func refName(ref string) string {
	if index := strings.LastIndex(ref, "/"); index >= 0 && index+1 < len(ref) {
		return strings.ReplaceAll(strings.ReplaceAll(ref[index+1:], "~1", "/"), "~0", "~")
	}
	return ref
}

func mergeHint(existing, hint string) string {
	if existing == "" {
		return hint
	}
	if existing == hint ||
		strings.HasPrefix(existing, hint+" (") ||
		strings.Contains(existing, fmt.Sprintf("(%s)", hint)) {
		return existing
	}
	return fmt.Sprintf("%s (%s)", existing, hint)
}

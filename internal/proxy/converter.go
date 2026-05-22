package proxy

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func anthropicToOpenAI(in anthropicRequest) openAIRequest {
	out := openAIRequest{
		Model:       in.Model,
		Stream:      in.Stream,
		MaxTokens:   in.MaxTokens,
		Temperature: in.Temperature,
		TopP:        in.TopP,
		Stop:        in.StopSequences,
	}
	if system := blocksToText(in.System); system != "" {
		out.Messages = append(out.Messages, openAIMessage{Role: "system", Content: system})
	}
	for _, msg := range in.Messages {
		out.Messages = append(out.Messages, anthropicMessageToOpenAI(msg)...)
	}
	for _, tool := range in.Tools {
		out.Tools = append(out.Tools, openAITool{
			Type: "function",
			Function: openAIFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}
	out.ToolChoice = convertToolChoice(in.ToolChoice)
	return out
}

func anthropicMessageToOpenAI(msg anthropicMsg) []openAIMessage {
	blocks, ok := msg.Content.([]any)
	if !ok {
		return []openAIMessage{{Role: normalizeRole(msg.Role), Content: blocksToText(msg.Content)}}
	}
	var messages []openAIMessage
	var textParts []string
	var toolCalls []toolCall
	var thinkingBlocks []string
	var imageParts []openAIMessage
	for _, block := range blocks {
		m, ok := block.(map[string]any)
		if !ok {
			continue
		}
		switch m["type"] {
		case "text":
			if text, _ := m["text"].(string); text != "" {
				textParts = append(textParts, text)
			}
		case "tool_use":
			id, _ := m["id"].(string)
			name, _ := m["name"].(string)
			args, _ := json.Marshal(m["input"])
			call := toolCall{ID: id, Type: "function"}
			call.Function.Name = name
			call.Function.Arguments = string(args)
			toolCalls = append(toolCalls, call)
		case "tool_result":
			if len(textParts) > 0 {
				messages = append(messages, openAIMessage{Role: "user", Content: strings.Join(textParts, "\n")})
				textParts = nil
			}
			if len(thinkingBlocks) > 0 {
				messages = append(messages, openAIMessage{Role: "assistant", ReasoningContent: strings.Join(thinkingBlocks, "\n")})
				thinkingBlocks = nil
			}
			if len(imageParts) > 0 {
				messages = append(messages, imageParts...)
				imageParts = nil
			}
			id, _ := m["tool_use_id"].(string)
			messages = append(messages, openAIMessage{Role: "tool", ToolCallID: id, Content: blocksToText(m["content"])})
		case "thinking":
			if text, _ := m["thinking"].(string); text != "" {
				thinkingBlocks = append(thinkingBlocks, text)
			}
		case "image":
			if source, _ := m["source"].(map[string]any); source != nil {
				stype, _ := source["type"].(string)
				switch stype {
				case "base64":
					mediaType, _ := source["media_type"].(string)
					data, _ := source["data"].(string)
					if mediaType != "" && data != "" {
						imageParts = append(imageParts, openAIMessage{
							Role:    normalizeRole(msg.Role),
							Content: fmt.Sprintf("![image](data:%s;base64,%s)", mediaType, data),
						})
					}
				case "url":
					url, _ := source["url"].(string)
					if url != "" {
						imageParts = append(imageParts, openAIMessage{
							Role:    normalizeRole(msg.Role),
							Content: fmt.Sprintf("![image](%s)", url),
						})
					}
				}
			}
			if len(imageParts) == 0 {
				textParts = append(textParts, "[image]")
			}
		}
	}
	if len(toolCalls) > 0 {
		content := strings.Join(textParts, "\n")
		reasoning := strings.Join(thinkingBlocks, "\n")
		messages = append(messages, openAIMessage{Role: "assistant", Content: content, ReasoningContent: reasoning, ToolCalls: toolCalls})
		return messages
	}
	if len(thinkingBlocks) > 0 {
		content := strings.Join(textParts, "\n")
		reasoning := strings.Join(thinkingBlocks, "\n")
		messages = append(messages, openAIMessage{Role: normalizeRole(msg.Role), Content: content, ReasoningContent: reasoning})
		return messages
	}
	if len(imageParts) > 0 {
		for _, ip := range imageParts {
			messages = append(messages, ip)
		}
		return messages
	}
	if len(textParts) > 0 {
		messages = append(messages, openAIMessage{Role: normalizeRole(msg.Role), Content: strings.Join(textParts, "\n")})
	}
	return messages
}

func openAIToAnthropic(in openAIResponse, model string) map[string]any {
	content := []map[string]any{}
	stopReason := "end_turn"
	if len(in.Choices) > 0 {
		choice := in.Choices[0]
		rc := reasoningText(choice.Message.ReasoningContent, choice.Message.ThinkingContent, choice.Message.Thinking, choice.Message.Reasoning, choice.Message.ReasoningDetails)
		if rc != "" {
			content = append(content, map[string]any{
				"type":     "thinking",
				"thinking": rc,
			})
		}
		if choice.Message.Content != "" {
			content = append(content, map[string]any{"type": "text", "text": choice.Message.Content})
		}
		for _, call := range choice.Message.ToolCalls {
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    fallbackToolID(call.ID),
				"name":  call.Function.Name,
				"input": parseJSONObj(call.Function.Arguments),
			})
		}
		stopReason = finishReason(choice.FinishReason, len(choice.Message.ToolCalls) > 0)
	}
	return map[string]any{
		"id":            firstNonEmpty(in.ID, "msg_ocgt_"+strconv.FormatInt(time.Now().UnixNano(), 36)),
		"type":          "message",
		"role":          "assistant",
		"model":         firstNonEmpty(in.Model, model),
		"content":       content,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]int{
			"input_tokens":  in.Usage.PromptTokens,
			"output_tokens": in.Usage.CompletionTokens,
		},
	}
}

func blocksToText(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []any:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			if m, ok := item.(map[string]any); ok {
				switch m["type"] {
				case "text":
					if text, _ := m["text"].(string); text != "" {
						parts = append(parts, text)
					}
				case "tool_result":
					if text := blocksToText(m["content"]); text != "" {
						parts = append(parts, text)
					}
				case "image":
					parts = append(parts, "[image]")
				}
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		return blocksToText([]any{x})
	default:
		return fmt.Sprint(x)
	}
}

func normalizeRole(role string) string {
	if role == "assistant" || role == "tool" || role == "system" {
		return role
	}
	return "user"
}

func convertToolChoice(choice any) any {
	m, ok := choice.(map[string]any)
	if !ok {
		return nil
	}
	switch m["type"] {
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "tool":
		name, _ := m["name"].(string)
		return map[string]any{"type": "function", "function": map[string]string{"name": name}}
	default:
		return nil
	}
}

func parseJSONObj(s string) map[string]any {
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err == nil {
		return out
	}
	return map[string]any{"arguments": s}
}

func finishReason(reason string, hasTool bool) string {
	if hasTool || reason == "tool_calls" {
		return "tool_use"
	}
	switch reason {
	case "length":
		return "max_tokens"
	case "stop":
		return "end_turn"
	default:
		return "end_turn"
	}
}

func fallbackToolID(id string) string {
	if id != "" {
		return id
	}
	return "toolu_ocgt_" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func reasoningText(values ...any) string {
	for _, value := range values {
		if text := reasoningTextValue(value); text != "" {
			return text
		}
	}
	return ""
}

func reasoningTextValue(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if text := reasoningTextValue(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		keys := []string{"reasoning_content", "thinking_content", "thinking", "reasoning", "content", "text", "summary"}
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			if text := reasoningTextValue(v[key]); text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return ""
}

func boundedThinkingPayload(thinking any, maxBudgetTokens int) any {
	if thinking == nil || maxBudgetTokens < 0 {
		return nil
	}
	if maxBudgetTokens == 0 {
		return thinking
	}
	switch v := thinking.(type) {
	case bool:
		if !v {
			return nil
		}
		return map[string]any{"type": "enabled", "budget_tokens": maxBudgetTokens}
	case string:
		if strings.EqualFold(v, "disabled") || strings.EqualFold(v, "false") {
			return nil
		}
		return map[string]any{"type": v, "budget_tokens": maxBudgetTokens}
	case map[string]any:
		out := make(map[string]any, len(v)+1)
		for key, value := range v {
			out[key] = value
		}
		if typ, _ := out["type"].(string); strings.EqualFold(typ, "disabled") {
			return out
		}
		out["budget_tokens"] = clampThinkingBudget(out["budget_tokens"], maxBudgetTokens)
		return out
	default:
		return map[string]any{"type": "enabled", "budget_tokens": maxBudgetTokens}
	}
}

func clampThinkingBudget(value any, maxBudgetTokens int) int {
	budget := intFromJSONNumber(value)
	if budget <= 0 || budget > maxBudgetTokens {
		return maxBudgetTokens
	}
	return budget
}

func intFromJSONNumber(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func singleJoin(base, path string) string {
	base = strings.TrimRight(base, "/")
	if base == "" {
		return path
	}
	return base + "/" + strings.TrimLeft(path, "/")
}

func isDeepSeekThinkingModel(model string) bool {
	lower := strings.ToLower(model)
	return strings.Contains(lower, "deepseek-v4") ||
		strings.Contains(lower, "deepseek-r1") ||
		strings.Contains(lower, "deepseek-reasoner") ||
		strings.Contains(lower, "ds-r1") ||
		strings.HasSuffix(lower, "/r1") ||
		strings.Contains(lower, "r1-") ||
		strings.Contains(lower, "reasoning")
}

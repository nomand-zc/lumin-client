package parser

import (
	"context"
	"encoding/json"

	"github.com/nomand-zc/lumin-client/log"
	"github.com/nomand-zc/lumin-client/providers"
)

// messageMetadataParser 处理 messageMetadataEvent 事件
// Kiro API 在流末尾返回此事件，包含精确的 token 用量信息
// 格式: { "tokenUsage": { "outputTokens": N, "totalTokens": N, "uncachedInputTokens": N, "cacheReadInputTokens": N, ... } }
type messageMetadataParser struct{}

func init() {
	Register(&messageMetadataParser{})
}

func (p *messageMetadataParser) MessageType() string { return MessageTypeEvent }
func (p *messageMetadataParser) EventType() string   { return EventTypeMessageMetadataEvent }

func (p *messageMetadataParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	return parseMetadataPayload(ctx, msg.Payload)
}

// metadataParser 处理 metadataEvent 事件（messageMetadataEvent 的别名格式）
type metadataParser struct{}

func init() {
	Register(&metadataParser{})
}

func (p *metadataParser) MessageType() string { return MessageTypeEvent }
func (p *metadataParser) EventType() string   { return EventTypeMetadataEvent }

func (p *metadataParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	return parseMetadataPayload(ctx, msg.Payload)
}

// supplementaryWebLinksParser 处理 supplementaryWebLinksEvent 事件
// 包含 inputTokens/outputTokens 信息
type supplementaryWebLinksParser struct{}

func init() {
	Register(&supplementaryWebLinksParser{})
}

func (p *supplementaryWebLinksParser) MessageType() string { return MessageTypeEvent }
func (p *supplementaryWebLinksParser) EventType() string   { return EventTypeSupplementaryWebLinks }

func (p *supplementaryWebLinksParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	return parseMetadataPayload(ctx, msg.Payload)
}

// usageEventParser 处理 usageEvent 事件
type usageEventParser struct{}

func init() {
	Register(&usageEventParser{})
}

func (p *usageEventParser) MessageType() string { return MessageTypeEvent }
func (p *usageEventParser) EventType() string   { return EventTypeUsageEvent }

func (p *usageEventParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	return parseMetadataPayload(ctx, msg.Payload)
}

// usageParser 处理 usage 事件
type usageParser struct{}

func init() {
	Register(&usageParser{})
}

func (p *usageParser) MessageType() string { return MessageTypeEvent }
func (p *usageParser) EventType() string   { return EventTypeUsage }

func (p *usageParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	return parseMetadataPayload(ctx, msg.Payload)
}

// tokenUsageData Kiro API 返回的精确 token 用量结构
type tokenUsageData struct {
	OutputTokens          float64 `json:"outputTokens"`
	TotalTokens           float64 `json:"totalTokens"`
	UncachedInputTokens   float64 `json:"uncachedInputTokens"`
	CacheReadInputTokens  float64 `json:"cacheReadInputTokens"`
	CacheWriteInputTokens float64 `json:"cacheWriteInputTokens"`
}

// parseMetadataPayload 解析元数据 payload 并提取 token 用量
func parseMetadataPayload(ctx context.Context, payload []byte) (*providers.Response, error) {
	// 尝试从嵌套格式解析
	var event map[string]json.RawMessage
	if err := json.Unmarshal(payload, &event); err != nil {
		log.Warnf("解析 metadata 载荷失败: %v", err)
		return nil, nil
	}

	// 尝试多种嵌套格式
	var metadata map[string]json.RawMessage
	if inner, ok := event["messageMetadataEvent"]; ok {
		json.Unmarshal(inner, &metadata)
	} else if inner, ok := event["metadataEvent"]; ok {
		json.Unmarshal(inner, &metadata)
	} else {
		metadata = event // 事件本身可能就是 metadata
	}

	if metadata == nil {
		return nil, nil
	}

	usage := &providers.Usage{}
	hasData := false

	// 优先检查 tokenUsage 嵌套对象（官方格式）
	if tokenUsageRaw, ok := metadata["tokenUsage"]; ok {
		var tu tokenUsageData
		if err := json.Unmarshal(tokenUsageRaw, &tu); err == nil {
			if tu.OutputTokens > 0 {
				usage.CompletionTokens = int(tu.OutputTokens)
				hasData = true
				log.Infof("metadata: 精确 outputTokens=%d", usage.CompletionTokens)
			}
			if tu.TotalTokens > 0 {
				usage.TotalTokens = int(tu.TotalTokens)
				hasData = true
			}
			// uncachedInputTokens + cacheReadInputTokens = 总 input tokens
			inputTokens := int(tu.UncachedInputTokens) + int(tu.CacheReadInputTokens)
			if inputTokens > 0 {
				usage.PromptTokens = inputTokens
				hasData = true
			}
			// 缓存读取 token 单独记录
			if tu.CacheReadInputTokens > 0 {
				usage.PromptTokensDetails.CacheReadTokens = int(tu.CacheReadInputTokens)
				usage.PromptTokensDetails.CachedTokens = int(tu.CacheReadInputTokens)
			}
		}
	}

	// 兜底：直接字段格式
	if !hasData {
		var directData struct {
			InputTokens  float64 `json:"inputTokens"`
			OutputTokens float64 `json:"outputTokens"`
			TotalTokens  float64 `json:"totalTokens"`
		}
		if err := json.Unmarshal(payload, &directData); err == nil {
			if directData.OutputTokens > 0 {
				usage.CompletionTokens = int(directData.OutputTokens)
				hasData = true
			}
			if directData.InputTokens > 0 {
				usage.PromptTokens = int(directData.InputTokens)
				hasData = true
			}
			if directData.TotalTokens > 0 {
				usage.TotalTokens = int(directData.TotalTokens)
				hasData = true
			}
		}
	}

	if !hasData {
		return nil, nil
	}

	return providers.NewResponse(ctx,
		providers.WithUsage(usage),
	), nil
}

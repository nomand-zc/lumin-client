package providers

import (
	"github.com/nomand-zc/lumin-client/credentials"
)

// GenerationConfig contains configuration for text generation.
type GenerationConfig struct {
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens *int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0 to 2.0).
	Temperature *float64 `json:"temperature,omitempty"`

	// TopP controls nucleus sampling (0.0 to 1.0).
	TopP *float64 `json:"top_p,omitempty"`

	// Stream indicates whether to stream the response.
	Stream bool `json:"stream"`

	// Stop sequences where the API will stop generating further tokens.
	Stop []string `json:"stop,omitempty"`

	// PresencePenalty penalizes new tokens based on their existing frequency.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// FrequencyPenalty penalizes new tokens based on their frequency in the text so far.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// ReasoningEffort limits the reasoning effort for reasoning models.
	// Supported values: "low", "medium", "high".
	// Only effective for OpenAI o-series models.
	ReasoningEffort *string `json:"reasoning_effort,omitempty"`

	// ThinkingEnabled enables thinking mode for Claude and Gemini models via OpenAI API.
	ThinkingEnabled *bool `json:"thinking_enabled,omitempty"`

	// ThinkingTokens controls the length of thinking for Claude and Gemini models via OpenAI API.
	ThinkingTokens *int `json:"thinking_tokens,omitempty"`
}

// Request is the request to the model.
type Request struct {
	// Credential 是本次请求使用的凭证信息。
	Credential credentials.Credential `json:"-"`

	// Header 允许调用者设置自定义的动态 Headers 信息，会在请求上游时附加到 HTTP 请求头中。
	Header map[string]string `json:"-"`

	// Metadata 允许协议层传递额外的元数据信息（如 tool_choice、response_format 等）
	// 这些信息可能被 provider 层消费来注入特定的行为。
	Metadata map[string]any `json:"-"`

	// Model is the model name to use for this request.
	// If empty, the provider's default model will be used.
	Model string `json:"model,omitempty"`

	// Messages is the conversation history.
	Messages []Message `json:"messages"`

	// GenerationConfig contains the generation parameters.
	GenerationConfig `json:"generation_config"`

	Tools []Tool `json:"tools,omitempty"` // Tools are not serialized, handled separately
}

// Tool is a tool that can be used by the model.
type Tool struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Parameters  Schema `json:"parameters"`
}

// Schema is the schema of the tool's arguments.
type Schema struct {
	//  Type Specifies the data type (e.g., "object", "array", "string", "number")
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	Required    []string `json:"required,omitempty"`
	// Properties of the arguments, each with its own schema
	Properties map[string]*Schema `json:"properties,omitempty"`
}

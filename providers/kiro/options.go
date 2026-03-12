package kiro

import (
	"maps"

	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/providers/tiktoken"
)

const (
	DefaultRegion = "us-east-1"
)

var defaultOptions = Options{
	url: "https://q.%s.amazonaws.com/generateAssistantResponse",
	headers: map[string]string{
		"Content-Type":    "application/json",
		"Accept":          "*/*",
		"amz-sdk-request": "attempt=1; max=3",

		// vibe
		"x-amzn-kiro-agent-mode": "vibe",

		// 遥测 opt-out
		"x-amzn-codewhisperer-optout": "true",

		// 注意: User-Agent 和 x-amz-user-agent 不在此处设置固定值，
		// 而是在 kiro_provider.go 中通过 FingerprintManager 动态生成。
	},
	defaultRegion: DefaultRegion,
}

func init() {
	tokenConter, err := tiktoken.New("claude-sonnet-4.6")
	if err != nil {
		panic(err)
	}
	defaultOptions.tokenConter = tokenConter
}

// Options contains the options for the client.
type Options struct {
	url           string
	headers       map[string]string
	defaultRegion string
	tokenConter   providers.TokenCounter
}

// Option is a function that sets an option.
type Option func(*Options)

// WithURL sets the URL.
func WithURL(url string) Option {
	return func(o *Options) {
		o.url = url
	}
}

// WithDefaultRegion sets the default region.
func WithDefaultRegion(region string) Option {
	return func(o *Options) {
		o.defaultRegion = region
	}
}

// WithHeader sets a single header key-value pair.
func WithHeader(key, value string) Option {
	return func(o *Options) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// WithHeaders merges the given headers into the options.
func WithHeaders(headers map[string]string) Option {
	return func(o *Options) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		maps.Copy(o.headers, headers)
	}
}

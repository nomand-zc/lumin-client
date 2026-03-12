package parser

import (
	"context"
	"encoding/json"

	"github.com/nomand-zc/lumin-client/log"
	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/utils"
)

// errorParser 处理 error 类型消息
type errorParser struct{}

func init() {
	Register(&errorParser{})
}

func (p *errorParser) MessageType() string { return MessageTypeError }
func (p *errorParser) EventType() string   { return "" }

func (p *errorParser) Parse(ctx context.Context, msg *StreamMessage, opts ...OptionFunc) (*providers.Response, error) {
	var errorData struct {
		Type    string `json:"__type"`
		Type2   string `json:"_type"`
		Message string `json:"message"`
	}

	if len(msg.Payload) > 0 {
		if err := json.Unmarshal(msg.Payload, &errorData); err != nil {
			log.Warnf("解析错误消息载荷失败: %v", err)
			errorData.Message = utils.Bytes2Str(msg.Payload)
		}
	}

	// 兼容 AWS 两种格式: __type 和 _type
	errType := errorData.Type
	if errType == "" {
		errType = errorData.Type2
	}

	return providers.NewResponse(ctx,
		providers.WithResponseError(&providers.ResponseError{
			Message: errorData.Message,
			Type:    "error",
			Code:    utils.ToPtr(errType),
		}),
	), nil
}

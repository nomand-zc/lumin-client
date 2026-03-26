package kiro

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/nomand-zc/lumin-client/credentials"
	kirocreds "github.com/nomand-zc/lumin-client/credentials/kiro"
	"github.com/nomand-zc/lumin-client/httpclient"
	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/usagerule"
	"github.com/nomand-zc/lumin-client/utils"
)

const (
	usageURL = "https://q.%s.amazonaws.com/getUsageLimits?isEmailRequired=true&origin=AI_EDITOR&resourceType=AGENTIC_REQUEST"
)

// kiroUsageResp 表示 getUsageLimits 接口的顶层响应
type kiroUsageResp struct {
	DaysUntilReset     int                  `json:"daysUntilReset"`
	NextDateReset      float64              `json:"nextDateReset"`
	UsageBreakdownList []usageBreakdownItem `json:"usageBreakdownList"`
	SubscriptionInfo   *subscriptionInfo    `json:"subscriptionInfo"`
	UserInfo           *userInfo            `json:"userInfo"`
}

// usageBreakdownItem 表示 usageBreakdownList 中的单个条目
type usageBreakdownItem struct {
	UsageLimit                int            `json:"usageLimit"`
	CurrentUsage              int            `json:"currentUsage"`
	CurrentUsageWithPrecision float64        `json:"currentUsageWithPrecision"`
	UsageLimitWithPrecision   float64        `json:"usageLimitWithPrecision"`
	ResourceType              string         `json:"resourceType"`
	Unit                      string         `json:"unit"`
	DisplayName               string         `json:"displayName"`
	DisplayNamePlural         string         `json:"displayNamePlural"`
	FreeTrialInfo             *freeTrialInfo `json:"freeTrialInfo"`
	OverageCap                int            `json:"overageCap"`
	OverageRate               float64        `json:"overageRate"`
}

// freeTrialInfo 表示免费试用信息
type freeTrialInfo struct {
	FreeTrialStatus           string  `json:"freeTrialStatus"`
	FreeTrialExpiry           float64 `json:"freeTrialExpiry"`
	UsageLimit                int     `json:"usageLimit"`
	CurrentUsage              int     `json:"currentUsage"`
	UsageLimitWithPrecision   float64 `json:"usageLimitWithPrecision"`
	CurrentUsageWithPrecision float64 `json:"currentUsageWithPrecision"`
}

// subscriptionInfo 表示订阅信息
type subscriptionInfo struct {
	SubscriptionTitle            string `json:"subscriptionTitle"`
	Type                         string `json:"type"`
	OverageCapability            string `json:"overageCapability"`
	UpgradeCapability            string `json:"upgradeCapability"`
	SubscriptionManagementTarget string `json:"subscriptionManagementTarget"`
}

// userInfo 表示用户信息
type userInfo struct {
	Email  string `json:"email"`
	UserID string `json:"userId"`
}

// convertToRules 将 kiroUsageResp 转换为 UsageRule 切片
func (r *kiroUsageResp) convertToRules() []*usagerule.UsageRule {
	if len(r.UsageBreakdownList) == 0 {
		return nil
	}

	rules := make([]*usagerule.UsageRule, 0, len(r.UsageBreakdownList))
	for _, item := range r.UsageBreakdownList {
		total := item.UsageLimitWithPrecision
		rule := &usagerule.UsageRule{
			SourceType:      usagerule.SourceTypeToken,
			TimeGranularity: usagerule.GranularityMonth,
			WindowSize:      1,
			Total:           total,
		}
		rules = append(rules, rule)
	}
	return rules
}

// convertToStats 将 kiroUsageResp 转换为 UsageStats 切片
func (r *kiroUsageResp) convertToStats() []*usagerule.UsageStats {
	if len(r.UsageBreakdownList) == 0 {
		return nil
	}

	statsList := make([]*usagerule.UsageStats, 0, len(r.UsageBreakdownList))
	for _, item := range r.UsageBreakdownList {
		total := item.UsageLimitWithPrecision
		used := item.CurrentUsageWithPrecision

		rule := &usagerule.UsageRule{
			SourceType:      usagerule.SourceTypeToken,
			TimeGranularity: usagerule.GranularityMonth,
			WindowSize:      1,
			Total:           total,
		}
		start, end := rule.CalculateWindowTime()
		stats := &usagerule.UsageStats{
			Rule:      rule,
			Used:      used,
			Remain:    total - used,
			StartTime: start,
			EndTime:   end,
		}
		statsList = append(statsList, stats)
	}
	return statsList
}

// Models 返回默认支持的模型列表
func (p *kiroProvider) Models(_ context.Context) ([]string, error) {
	models := make([]string, 0, len(ModelList))
	for _, k := range ModelList {
		models = append(models, k)
	}
	return models, nil
}

// ListModels 获取当前凭证支持的模型列表
// kiro 不提供动态模型列表接口，直接返回默认模型列表
func (p *kiroProvider) ListModels(ctx context.Context, _ credentials.Credential) ([]string, error) {
	return p.Models(ctx)
}

// DefaultUsageRules 获取供应商默认的用量规则列表
func (p *kiroProvider) DefaultUsageRules(_ context.Context) ([]*usagerule.UsageRule, error) {
	return []*usagerule.UsageRule{
		{
			SourceType:      usagerule.SourceTypeToken,
			TimeGranularity: usagerule.GranularityMonth,
			WindowSize:      1,
			Total:           50,
		},
	}, nil
}

// GetUsageRules 获取当前凭证的用量规则列表
func (p *kiroProvider) GetUsageRules(ctx context.Context, creds credentials.Credential) ([]*usagerule.UsageRule, error) {
	result, err := p.fetchUsageResp(ctx, creds)
	if err != nil {
		return nil, err
	}
	return result.convertToRules(), nil
}

// GetUsageStats 获取当前凭证的用量统计信息
func (p *kiroProvider) GetUsageStats(ctx context.Context, creds credentials.Credential) ([]*usagerule.UsageStats, error) {
	result, err := p.fetchUsageResp(ctx, creds)
	if err != nil {
		return nil, err
	}

	// 更新凭证中的用户信息
	if result != nil && result.UserInfo != nil {
		if kiroCreds, ok := creds.(*kirocreds.Credential); ok {
			if kiroCreds.User.ID == "" {
				kiroCreds.User.ID = result.UserInfo.UserID
			}
			if kiroCreds.User.Email == "" {
				kiroCreds.User.Email = result.UserInfo.Email
			}
		}
	}
	return result.convertToStats(), nil
}

// fetchUsageResp 发送请求并解析用量响应
func (p *kiroProvider) fetchUsageResp(ctx context.Context, creds credentials.Credential) (*kiroUsageResp, error) {
	resp, err := p.send(ctx, creds)
	if err != nil {
		return nil, errors.Annotate(err, "send get usage limits request failed")
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Annotate(err, "read get usage limits response failed")
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return nil, &providers.HTTPError{
			ErrorType:     providers.ErrorTypeRateLimit,
			ErrorCode:     providers.ErrorCodeRateLimit,
			Message:       "rate limit exceeded",
			RawStatusCode: resp.StatusCode,
			RawBody:       respBody,
		}
	default:
		if resp.StatusCode != http.StatusOK {
			return nil, &providers.HTTPError{
				ErrorType:     providers.ErrorTypeServerError,
				ErrorCode:     resp.StatusCode,
				Message:       fmt.Sprintf("get usage limits failed, status=%d", resp.StatusCode),
				RawStatusCode: resp.StatusCode,
				RawBody:       respBody,
			}
		}
	}

	var result kiroUsageResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, errors.Annotatef(err, "parse get usage limits response failed, status=%d, body=%s",
			resp.StatusCode, utils.Bytes2Str(respBody))
	}
	return &result, nil
}

func (p *kiroProvider) send(ctx context.Context, creds credentials.Credential) (*http.Response, error) {
	kiroCreds, ok := creds.(*kirocreds.Credential)
	if !ok {
		return nil, errors.New("invalid credentials type")
	}
	rawURL := fmt.Sprintf(usageURL, kiroCreds.Region)
	if kiroCreds.AuthMethod == kirocreds.AuthMethodSocial && kiroCreds.ProfileArn != "" {
		rawURL += "&profileArn=" + kiroCreds.ProfileArn
	}
	req, err := http.NewRequestWithContext(httpclient.EnablePrintRespBody(ctx),
		http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, errors.Annotate(err, "create get usage limits request failed")
	}
	for k, v := range p.options.headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", "Bearer "+kiroCreds.AccessToken)
	req.Header.Set("amz-sdk-invocation-id", uuid.NewString())

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, errors.Annotate(err, "get usage limits request failed")
	}
	return resp, nil
}

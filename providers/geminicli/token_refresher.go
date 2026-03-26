package geminicli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	jujuerrors "github.com/juju/errors"
	"github.com/nomand-zc/lumin-client/credentials"
	geminicreds "github.com/nomand-zc/lumin-client/credentials/geminicli"
	"github.com/nomand-zc/lumin-client/httpclient"
	"github.com/nomand-zc/lumin-client/providers"
)

// Refresh 刷新 GeminiCLI 的 OAuth2 令牌，直接修改入参 creds 中的字段。
// 使用标准 Google OAuth2 refresh_token 流程向 token_uri 发送请求获取新的 access_token。
func (p *geminicliProvider) Refresh(ctx context.Context, creds credentials.Credential) error {
	geminiCreds, ok := creds.(*geminicreds.Credential)
	if !ok {
		return fmt.Errorf("invalid credentials type, expected *geminicreds.Credential")
	}

	return p.refreshOAuth2Token(ctx, geminiCreds)
}

// tokenRefreshResp Google OAuth2 token 刷新响应
type tokenRefreshResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // Token 有效期（秒）
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
	IDToken     string `json:"id_token,omitempty"`

	// 错误字段
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// refreshOAuth2Token 使用 refresh_token 通过 Google OAuth2 端点刷新 access_token
func (p *geminicliProvider) refreshOAuth2Token(ctx context.Context, creds *geminicreds.Credential) error {
	tokenURI := creds.TokenURI
	if tokenURI == "" {
		tokenURI = geminicreds.DefaultTokenURI
	}

	clientID := creds.ClientID
	if clientID == "" {
		clientID = geminicreds.DefaultClientID
	}

	clientSecret := creds.ClientSecret
	if clientSecret == "" {
		clientSecret = geminicreds.DefaultClientSecret
	}

	// 构建 application/x-www-form-urlencoded 请求体
	formData := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"refresh_token": {creds.RefreshToken},
		"grant_type":    {"refresh_token"},
	}

	req, err := http.NewRequestWithContext(httpclient.EnablePrintRespBody(ctx),
		http.MethodPost, tokenURI, strings.NewReader(formData.Encode()))
	if err != nil {
		return jujuerrors.Annotate(err, "create token refresh request failed")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return jujuerrors.Annotatef(err, "token refresh request failed, url=%s", tokenURI)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return jujuerrors.Annotate(err, "read token refresh response failed")
	}

	// 处理 HTTP 错误状态码
	if resp.StatusCode != http.StatusOK {
		var errResp tokenRefreshResp
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Error != "" {
			if errResp.Error == "invalid_grant" {
				return providers.ErrInvalidGrant
			}
			return newHTTPError(resp.StatusCode,
				fmt.Sprintf("token refresh failed: %s - %s", errResp.Error, errResp.ErrorDescription),
				respBody)
		}
		return newHTTPErrorf(resp.StatusCode, respBody,
			"token refresh failed, status=%d", resp.StatusCode)
	}

	// 解析成功响应
	var result tokenRefreshResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return jujuerrors.Annotatef(err, "parse token refresh response failed, status=%d, body=%s",
			resp.StatusCode, string(respBody))
	}

	if result.Error != "" {
		if result.Error == "invalid_grant" {
			return providers.ErrInvalidGrant
		}
		return fmt.Errorf("token refresh error: %s - %s", result.Error, result.ErrorDescription)
	}

	// 更新凭证字段
	creds.AccessToken = result.AccessToken
	creds.Token = result.AccessToken
	if result.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		creds.ExpiresAt = &expiresAt
	}
	// 重置 raw 缓存，使 ToMap() 返回最新数据
	creds.ResetRaw()

	return nil
}

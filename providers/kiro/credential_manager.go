package kiro

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	jujuerrors "github.com/juju/errors"
	"github.com/nomand-zc/lumin-client/credentials"
	kirocreds "github.com/nomand-zc/lumin-client/credentials/kiro"
	"github.com/nomand-zc/lumin-client/httpclient"
	"github.com/nomand-zc/lumin-client/providers"
	"github.com/nomand-zc/lumin-client/utils"
)

const (
	socailRefreshURL = "https://prod.%s.auth.desktop.kiro.dev/refreshToken"
	idcRefreshURL    = "https://oidc.%s.amazonaws.com/token"

	authMethodSocial = "social"
)

// Refresh 刷新令牌，直接修改入参 creds 中的字段。
func (r *kiroProvider) Refresh(ctx context.Context, creds credentials.Credential) error {
	kiroCreds, ok := creds.(*kirocreds.Credential)
	if !ok {
		return fmt.Errorf("invalid credentials type")
	}

	if kiroCreds.AuthMethod == authMethodSocial {
		return r.refreshSocialToken(ctx, kiroCreds)
	}

	return r.refreshIDCToken(ctx, kiroCreds)
}

type tokenRefreshResp struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"` // 刷新后可能返回新的 refreshToken
	ExpiresIn    int    `json:"expiresIn"`    // Token 有效期（秒），用于计算 expiresAt
	ProfileArn   string `json:"profileArn"`

	Error string `json:"error"` // 错误码
}

// doRefreshRequest 发送 token 刷新 HTTP 请求，返回解析后的响应结果
func (r *kiroProvider) doRefreshRequest(ctx context.Context, refreshURL string, reqBody any) (*tokenRefreshResp, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, jujuerrors.Annotate(err, "marshal refresh request failed")
	}

	req, err := http.NewRequestWithContext(httpclient.EnablePrintRespBody(ctx),
		http.MethodPost, refreshURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, jujuerrors.Annotate(err, "create refresh request failed")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, jujuerrors.Annotatef(err, "refresh request failed, url=%s", refreshURL)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, jujuerrors.Annotate(err, "read refresh response failed")
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return nil, &providers.HTTPError{
			ErrorType:     providers.ErrorTypeRateLimit,
			ErrorCode:     resp.StatusCode,
			Message:       "token refresh rate limit",
			RawStatusCode: resp.StatusCode,
			RawBody:       respBody,
		}
	default:
		if resp.StatusCode != http.StatusOK {
			return nil, providers.ErrInvalidGrant
		}
	}

	var result tokenRefreshResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, jujuerrors.Annotatef(err, "parse refresh response failed, status=%d, body=%s",
			resp.StatusCode, utils.Bytes2Str(respBody))
	}

	return &result, nil
}

func (r *kiroProvider) refreshSocialToken(ctx context.Context, creds *kirocreds.Credential) error {
	refreshURL := fmt.Sprintf(socailRefreshURL, creds.Region)
	reqBody := map[string]string{"refreshToken": creds.RefreshToken}

	result, err := r.doRefreshRequest(ctx, refreshURL, reqBody)
	if err != nil {
		return jujuerrors.Annotate(err, "kiro social token refresh failed")
	}

	creds.AccessToken = result.AccessToken
	creds.RefreshToken = result.RefreshToken
	creds.ProfileArn = result.ProfileArn
	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	creds.ExpiresAt = &expiresAt

	return nil
}

func (r *kiroProvider) refreshIDCToken(ctx context.Context, creds *kirocreds.Credential) error {
	refreshURL := fmt.Sprintf(idcRefreshURL, creds.IDCRegion)
	reqBody := map[string]string{
		"refreshToken": creds.RefreshToken,
		"clientId":     creds.ClientID,
		"clientSecret": creds.ClientSecret,
		"grantType":    "refresh_token",
	}

	result, err := r.doRefreshRequest(ctx, refreshURL, reqBody)
	if err != nil {
		return jujuerrors.Annotate(err, "kiro IDC token refresh failed")
	}

	creds.AccessToken = result.AccessToken
	creds.RefreshToken = result.RefreshToken
	creds.ProfileArn = result.ProfileArn
	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	creds.ExpiresAt = &expiresAt

	return nil
}

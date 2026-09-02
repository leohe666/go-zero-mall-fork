// Package wechatx 封装微信小程序服务端 API：
//   - GetPhoneNumber: 用 getPhoneNumber 组件返回的一次性 code 换取用户手机号
//     （POST /wxa/business/getuserphonenumber，需 client_credential access_token）
//
// AppSecret 不应硬编码在配置/代码中：通过环境变量注入到 etc/*.yaml（如 ${WECHAT_APP_SECRET}）。
package wechatx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Config 微信小程序服务端配置，与 etc/*.yaml 中的 Wechat 段对应。
type Config struct {
	AppId     string `json:"AppId"`
	AppSecret string `json:"AppSecret"`
}

const (
	tokenURL    = "https://api.weixin.qq.com/cgi-bin/token"
	phoneNumber = "https://api.weixin.qq.com/wxa/business/getuserphonenumber"
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}

	// tokenMu 保护 tokenCache；微信 access_token 与 appid 强绑定，
	// SaaS 多商户下必须按 appid 隔离缓存，禁止跨商户复用。
	tokenMu    sync.Mutex
	tokenCache = map[string]tokenEntry{}
)

type tokenEntry struct {
	value   string
	expires time.Time
}

// accessToken 获取 client_credential access_token（2 小时有效，按 appid 进程内缓存复用）。
func accessToken(ctx context.Context, cfg Config) (string, error) {
	tokenMu.Lock()
	defer tokenMu.Unlock()

	if e, ok := tokenCache[cfg.AppId]; ok && e.value != "" && time.Now().Before(e.expires) {
		return e.value, nil
	}

	q := url.Values{}
	q.Set("grant_type", "client_credential")
	q.Set("appid", cfg.AppId)
	q.Set("secret", cfg.AppSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL+"?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("wechat token request error: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("parse wechat token error: %w, body: %s", err, string(body))
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("wechat token error: errcode=%d errmsg=%s", tr.ErrCode, tr.ErrMsg)
	}

	tokenCache[cfg.AppId] = tokenEntry{
		value:   tr.AccessToken,
		expires: time.Now().Add(time.Duration(tr.ExpiresIn-300) * time.Second), // 提前 5 分钟过期
	}
	return tr.AccessToken, nil
}

// PhoneInfo 微信返回的手机号信息。
type PhoneInfo struct {
	PhoneNumber     string `json:"phoneNumber"`
	PurePhoneNumber string `json:"purePhoneNumber"`
	CountryCode     string `json:"countryCode"`
}

// GetPhoneNumber 用 getPhoneNumber 组件返回的一次性 code 换取用户手机号。
// 返回纯手机号（如 13800138000），换取失败返回 error。
func GetPhoneNumber(ctx context.Context, cfg Config, code string) (string, error) {
	if cfg.AppId == "" || cfg.AppSecret == "" {
		return "", fmt.Errorf("wechat app credentials not configured")
	}
	if code == "" {
		return "", fmt.Errorf("phone code is empty")
	}

	token, err := accessToken(ctx, cfg)
	if err != nil {
		return "", err
	}

	payload := fmt.Sprintf(`{"code":%q}`, code)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		phoneNumber+"?access_token="+url.QueryEscape(token), strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("wechat phone request error: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var pr struct {
		ErrCode   int       `json:"errcode"`
		ErrMsg    string    `json:"errmsg"`
		PhoneInfo PhoneInfo `json:"phone_info"`
	}
	if err := json.Unmarshal(body, &pr); err != nil {
		return "", fmt.Errorf("parse wechat phone response error: %w, body: %s", err, string(body))
	}
	if pr.ErrCode != 0 {
		return "", fmt.Errorf("wechat getphonenumber error: errcode=%d errmsg=%s", pr.ErrCode, pr.ErrMsg)
	}
	if pr.PhoneInfo.PurePhoneNumber == "" {
		return "", fmt.Errorf("wechat getphonenumber returned empty phone, body: %s", string(body))
	}
	return pr.PhoneInfo.PurePhoneNumber, nil
}

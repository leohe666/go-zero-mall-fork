// Package casdoorx 封装 Casdoor (SaaS 身份认证平台) 的客户端能力：
//   - ExchangeMiniProgramCode: 将微信小程序 wx.login() 的 code 交给 Casdoor 换取 access_token
//     （Casdoor 负责与微信服务器换取 openid 并自动创建/更新用户）
//   - ParseToken: 校验 Casdoor 签发的 JWT（使用 Casdoor 应用的证书公钥）
//   - MockOpenId: 本地开发模式（无真实微信凭据）下模拟 openid
package casdoorx

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// Config Casdoor 配置，与各服务 etc/*.yaml 中的 Casdoor 段对应。
// 基于 SaaS 模式：Endpoint 指向 Casdoor 实例（云 SaaS 或自建），
// 每个租户 = Casdoor 中的一个 organization，本应用固定使用 OrganizationName/ApplicationName。
type Config struct {
	Endpoint         string `json:"Endpoint"`
	ClientId         string `json:"ClientId"`
	ClientSecret     string `json:"ClientSecret"`
	Certificate      string `json:"Certificate"`
	OrganizationName string `json:"OrganizationName"`
	ApplicationName  string `json:"ApplicationName"`
	// MockMiniProgram 仅本地开发使用：为 true 时跳过 Casdoor→微信的 code 交换，
	// 直接用 code 生成确定性 openid，便于无真实微信凭据时联调小程序登录流程。
	MockMiniProgram bool `json:"MockMiniProgram"`
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// MiniProgramTokenResponse Casdoor /api/login/oauth/access_token 的响应
// 注意：Casdoor 成功响应使用驼峰字段（accessToken/tokenType/...），
// 错误响应使用标准 OAuth 下划线字段（error/error_description），这里两种都兼容。
type MiniProgramTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`

	// Casdoor 新版成功响应的驼峰字段
	AccessTokenCamel  string `json:"accessToken"`
	RefreshTokenCamel string `json:"refreshToken"`
	TokenTypeCamel    string `json:"tokenType"`
	ExpiresInCamel    int    `json:"expiresIn"`
}

// EffectiveAccessToken 返回实际有效的 access_token（兼容驼峰/下划线两种字段命名）
func (r *MiniProgramTokenResponse) EffectiveAccessToken() string {
	if r.AccessToken != "" {
		return r.AccessToken
	}
	return r.AccessTokenCamel
}

// ExchangeMiniProgramCode 微信小程序登录：把 wx.login() 的 code 交给 Casdoor 换取 access_token。
// 请求体为表单：tag=wechat_miniprogram（必须）+ client_id + code（+ 可选 username/avatar）。
func ExchangeMiniProgramCode(ctx context.Context, cfg Config, code, username, avatar string) (*MiniProgramTokenResponse, error) {
	form := url.Values{}
	form.Set("tag", "wechat_miniprogram")
	form.Set("client_id", cfg.ClientId)
	form.Set("code", code)
	if username != "" {
		form.Set("username", username)
	}
	if avatar != "" {
		form.Set("avatar", avatar)
	}

	endpoint := strings.TrimSuffix(cfg.Endpoint, "/") + "/api/login/oauth/access_token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call casdoor access_token error: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var tokenResp MiniProgramTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse casdoor response error: %w, body: %s", err, string(body))
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("casdoor error: %s: %s", tokenResp.Error, tokenResp.Description)
	}
	if tokenResp.EffectiveAccessToken() == "" {
		return nil, fmt.Errorf("casdoor returned empty access_token, body: %s", string(body))
	}
	return &tokenResp, nil
}

// Claims Casdoor JWT 声明（用户信息嵌入在 claims 中）
type Claims struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Id    string `json:"id"`
}

// ParseToken 校验 Casdoor 签发的 JWT（RS256/ES256），返回其中携带的用户声明。
// certificate 为 Casdoor 应用证书的公钥 PEM（x509 证书或 PUBLIC KEY 均可）。
func ParseToken(ctx context.Context, cfg Config, accessToken string) (*Claims, error) {
	casdoorsdk.InitConfig(
		cfg.Endpoint,
		cfg.ClientId,
		cfg.ClientSecret,
		cfg.Certificate,
		cfg.OrganizationName,
		cfg.ApplicationName,
	)

	parsed, err := casdoorsdk.ParseJwtToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("parse casdoor jwt error: %w", err)
	}
	return &Claims{
		Owner: parsed.Owner,
		Name:  parsed.Name,
		Id:    parsed.Id,
	}, nil
}

// MockOpenId 本地开发模式：由 code 生成确定性 openid（模拟微信 jscode2session 返回）。
// 相同 code 始终得到相同 openid，便于重复登录测试；生产环境请勿开启 MockMiniProgram。
func MockOpenId(code string) string {
	sum := sha1.Sum([]byte("mall-mock:" + code))
	return "mock-" + hex.EncodeToString(sum[:8])
}

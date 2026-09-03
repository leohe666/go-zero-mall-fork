// Package casdoorx 封装 Casdoor (SaaS 身份认证平台) 的客户端能力：
//   - ExchangeMiniProgramCode: 将微信小程序 wx.login() 的 code 交给 Casdoor 换取 access_token
//     （Casdoor 负责与微信服务器换取 openid 并自动创建/更新用户）
//   - ParseToken: 校验 Casdoor 签发的 JWT（使用对应应用的证书公钥）
//   - UpdateUserPhone: 用应用 client_credentials（admin 上下文）把手机号写回 Casdoor
//     （用户自更新 token 受 ID/Country code/User type 等 ModifyRule 守卫限制，无法写回 cc 为空的用户）
//
// 多商户模式下每个商户携带自己的 clientId + clientSecret + 证书，操作均按传入参数隔离。
package casdoorx

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

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// Config Casdoor 配置，与各服务 etc/*.yaml 中的 Casdoor 段对应。
// 基于 SaaS 模式：Endpoint 指向 Casdoor 实例（云 SaaS 或自建）。
type Config struct {
	Endpoint         string `json:"Endpoint"`
	ClientId         string `json:"ClientId"`
	ClientSecret     string `json:"ClientSecret"`
	Certificate      string `json:"Certificate"`
	OrganizationName string `json:"OrganizationName"`
	ApplicationName  string `json:"ApplicationName"`
}

var httpClient = &http.Client{Timeout: 15 * time.Second}

// client_credentials token 进程内缓存（clientId+clientSecret 换取的 admin 上下文 token）。
var (
	ccMu      sync.Mutex
	ccToken   string
	ccExpires time.Time
)

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

// Claims Casdoor JWT 声明（用户信息嵌入在 claims 中）。Id 为稳定唯一关联键。
type Claims struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
	Id    string `json:"id"`
}

// ParseToken 校验 Casdoor 签发的 JWT（RS256/ES256），返回其中携带的用户声明。
// certificate 为对应 Casdoor 应用证书的公钥 PEM（x509 证书或 PUBLIC KEY 均可）。
func ParseToken(ctx context.Context, cfg Config, accessToken string) (*Claims, error) {
	client := casdoorsdk.NewClient(
		cfg.Endpoint,
		cfg.ClientId,
		cfg.ClientSecret,
		cfg.Certificate,
		cfg.OrganizationName,
		cfg.ApplicationName,
	)

	parsed, err := client.ParseJwtToken(accessToken)
	if err != nil {
		return nil, fmt.Errorf("parse casdoor jwt error: %w", err)
	}

	// 旧版本 Casdoor 在 JWT 里没有 id 时，从 userinfo 接口补充
	claims := &Claims{
		Owner: parsed.Owner,
		Name:  parsed.Name,
		Id:    parsed.Id,
	}
	if claims.Id == "" {
		u, err := client.WithAccessToken(accessToken).GetAccount()
		if err == nil && u != nil {
			claims.Id = u.Id
			if claims.Name == "" {
				claims.Name = u.Name
			}
		}
	}
	if claims.Id == "" {
		return nil, fmt.Errorf("casdoor jwt has no stable user id (id claim)")
	}
	return claims, nil
}

// UpdateUserPhone 用应用级 client_credentials（admin 上下文）把手机号写回 Casdoor 用户。
//
// 为什么不用用户自更新 access_token：Casdoor 3.163.0 对普通用户更新有一连串
// accountItems ModifyRule 守卫（ID=Immutable、Country code=Admin、User type=Admin…），
// 微信自动创建的用户 countryCode 为空，写 phone 时服务端归一化连带写 countryCode，
// 且 SDK 全量序列化 User 结构体导致零值字段覆盖真实值，普通用户 token 会被拒
// （"The ID is immutable"/"Only admin can modify the Country code"/"Only admin can modify the User type"）。
// client_credentials 的 admin token 一次性绕过所有 ModifyRule，是官方后端管理 API 模式。
//
// clientSecret 由网关从 merchant 表经 GetMerchant 获取（user rpc 用主密钥解密），
// 不落配置文件；token 进程内缓存、提前 5 分钟过期。
//
// userId 必须传 Casdoor 用户的稳定 Id（UUID）且与目标一致：
// ID 字段 modifyRule=Immutable，body 中缺失或不一致会报 "The ID is immutable"。
func UpdateUserPhone(ctx context.Context, cfg Config, userId, owner, name, phone, countryCode string) error {
	if phone == "" {
		return nil
	}
	if cfg.ClientSecret == "" {
		return fmt.Errorf("casdoor client secret not configured for phone writeback")
	}

	// 1) client_credentials 换 admin 上下文 token
	token, err := oauthClientCredentialsToken(ctx, cfg)
	if err != nil {
		return err
	}

	// 2) 以 admin 身份更新 phone + countryCode
	client := casdoorsdk.NewClient(
		cfg.Endpoint,
		cfg.ClientId,
		cfg.ClientSecret,
		cfg.Certificate,
		cfg.OrganizationName,
		cfg.ApplicationName,
	).WithAccessToken(token)

	u := &casdoorsdk.User{
		Owner:       owner,
		Name:        name,
		Id:          userId,
		Phone:       phone,
		CountryCode: countryCode,
	}
	columns := []string{"phone"}
	if countryCode != "" {
		columns = append(columns, "countryCode")
	}
	affected, err := client.UpdateUserForColumns(u, columns)
	if err != nil {
		return fmt.Errorf("update casdoor user phone error: %w", err)
	}
	if !affected {
		return fmt.Errorf("update casdoor user phone returned no affected rows")
	}
	return nil
}

// oauthClientCredentialsToken 用应用 clientId+clientSecret 换取 Casdoor access_token
// （grant_type=client_credentials，admin 上下文）。token 进程内缓存、提前 5 分钟过期。
func oauthClientCredentialsToken(ctx context.Context, cfg Config) (string, error) {
	ccMu.Lock()
	defer ccMu.Unlock()

	if ccToken != "" && time.Now().Before(ccExpires) {
		return ccToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", cfg.ClientId)
	form.Set("client_secret", cfg.ClientSecret)

	endpoint := strings.TrimSuffix(cfg.Endpoint, "/") + "/api/login/oauth/access_token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call casdoor client_credentials error: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tr MiniProgramTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("parse casdoor client_credentials error: %w, body: %s", err, string(body))
	}
	if tr.Error != "" {
		return "", fmt.Errorf("casdoor client_credentials error: %s: %s", tr.Error, tr.Description)
	}
	token := tr.EffectiveAccessToken()
	if token == "" {
		return "", fmt.Errorf("casdoor client_credentials returned empty token, body: %s", string(body))
	}

	ttl := tr.ExpiresIn
	if ttl == 0 {
		ttl = tr.ExpiresInCamel
	}
	if ttl <= 0 {
		ttl = 7200 // 默认 2 小时
	}
	ccToken = token
	ccExpires = time.Now().Add(time.Duration(ttl-300) * time.Second)
	return token, nil
}

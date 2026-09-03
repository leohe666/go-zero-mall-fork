// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"errors"
	"time"

	"mall/common/casdoorx"
	"mall/common/jwtx"
	"mall/common/wechatx"
	"mall/service/gateway/api/internal/svc"
	"mall/service/gateway/api/internal/types"
	"mall/service/user/rpc/types/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type MpLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMpLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MpLoginLogic {
	return &MpLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// MpLogin 微信小程序登录（SaaS 多商户，全真实链路）：
//  1. 小程序带 merchantId + wx.login() code + phoneCode 调用本接口；
//  2. 网关按 merchantId 向 user rpc 取该商户的 Casdoor 租户配置 + 微信小程序凭据
//     （AppSecret 在 user rpc 内解密，凭据不落配置文件/网关进程）；
//  3. 用该商户的 Casdoor clientId 把 code 交给 Casdoor 换 access_token（Casdoor 与微信换 openid，
//     自动创建/更新该商户组织下的 Casdoor 用户）；
//  4. 校验 Casdoor JWT，取出稳定唯一关联键 casdoorId；
//  5. 用该商户的微信凭据 + phoneCode 向微信换取真实手机号；
//  6. 用用户自己的 Casdoor JWT 把手机号写回 Casdoor 用户信息（无需 clientSecret）；
//  7. 以 (merchantId, casdoorId) 落地本地 user 表（首次自动注册），签发商城 JWT。
func (l *MpLoginLogic) MpLogin(req *types.MpLoginRequest) (resp *types.MpLoginResponse, err error) {
	if req.MerchantId <= 0 {
		return nil, errors.New("merchantId is required")
	}
	if req.Code == "" {
		return nil, errors.New("mp login code is empty")
	}
	// 手机号必填：小程序 getPhoneNumber 授权是真实登录的必填环节
	if req.PhoneCode == "" {
		return nil, errors.New("phone authorization required: phoneCode is empty")
	}

	// 1) 按商户取租户配置（user rpc 内解密微信 AppSecret）
	m, err := l.svcCtx.UserRpc.GetMerchant(l.ctx, &user.GetMerchantRequest{Id: req.MerchantId})
	if err != nil {
		l.Logger.Errorf("get merchant %d error: %v", req.MerchantId, err)
		return nil, err
	}
	if m.Status != 1 || m.CasdoorEndpoint == "" || m.WxAppId == "" {
		return nil, errors.New("商户未启用或配置不完整")
	}

	cfg := casdoorx.Config{
		Endpoint:         m.CasdoorEndpoint,
		ClientId:         m.CasdoorClientId,
		OrganizationName: m.CasdoorOrg,
		ApplicationName:  m.CasdoorApp,
		Certificate:      m.CasdoorCertPem,
	}
	wx := wechatx.Config{AppId: m.WxAppId, AppSecret: m.WxAppSecret}

	// 2) 用 code 向该商户的 Casdoor 换取 access_token
	tokenResp, err := casdoorx.ExchangeMiniProgramCode(l.ctx, cfg, req.Code, req.Username, req.Avatar)
	if err != nil {
		l.Logger.Errorf("exchange mini program code error: %v", err)
		return nil, err
	}
	accessToken := tokenResp.EffectiveAccessToken()

	// 3) 校验 Casdoor JWT，得到稳定关联键 casdoorId
	claims, err := casdoorx.ParseToken(l.ctx, cfg, accessToken)
	if err != nil {
		l.Logger.Errorf("parse casdoor token error: %v", err)
		return nil, err
	}
	l.Logger.Infof("merchant %d casdoor user %s(%s) authorized via mini program", req.MerchantId, claims.Name, claims.Id)

	// 4) 手机号：用该商户的微信凭据 + 一次性 phoneCode 换取真实手机号
	mobile, err := wechatx.GetPhoneNumber(l.ctx, wx, req.PhoneCode)
	if err != nil {
		l.Logger.Errorf("get wechat phone number error: %v", err)
		return nil, errors.New("获取手机号失败: " + err.Error())
	}
	l.Logger.Infof("mini program user %s phone %s", claims.Name, maskPhone(mobile))

	// 5) 用用户自己的 Casdoor JWT 把手机号+国家码写回 Casdoor（失败不阻断本地登录）。
	//    需 mall 组织 Country code modifyRule=Self（已在控制台配置），
	//    body 带稳定 Id（ID 字段 Immutable，缺失会报 "The ID is immutable"）。
	if err := casdoorx.UpdateUserPhone(l.ctx, cfg, accessToken, claims.Id, claims.Owner, claims.Name, mobile, "CN"); err != nil {
		l.Logger.Errorf("update casdoor user phone error: %v", err)
	} else {
		l.Logger.Infof("casdoor user %s phone synced to casdoor", claims.Id)
	}

	// 6) 以 (merchantId, casdoorId) 落地本地用户（首次自动创建）
	res, err := l.svcCtx.UserRpc.LoginByCasdoor(l.ctx, &user.LoginByCasdoorRequest{
		MerchantId:   req.MerchantId,
		CasdoorId:    claims.Id,
		CasdoorName:  claims.Name,
		Mobile:       mobile,
	})
	if err != nil {
		l.Logger.Errorf("login by casdoor error: %v", err)
		return nil, err
	}

	// 7) 签发商城自身 JWT
	now := time.Now().Unix()
	accessExpire := l.svcCtx.Config.Auth.AccessExpire
	accessToken, err = jwtx.GetToken(l.svcCtx.Config.Auth.AccessSecret, now, accessExpire, res.Id)
	if err != nil {
		return nil, err
	}

	return &types.MpLoginResponse{
		AccessToken:  accessToken,
		AccessExpire: now + accessExpire,
		UserId:       res.Id,
		MerchantId:   res.MerchantId,
		CasdoorId:    res.CasdoorId,
		CasdoorName:  res.CasdoorName,
		Mobile:       res.Mobile,
	}, nil
}

// maskPhone 手机号脱敏打日志（138****8000）
func maskPhone(p string) string {
	if len(p) < 7 {
		return "****"
	}
	return p[:3] + "****" + p[len(p)-4:]
}
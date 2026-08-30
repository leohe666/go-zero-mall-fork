// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"mall/common/casdoorx"
	"mall/common/jwtx"
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

// MpLogin 微信小程序登录（Casdoor SaaS 模式）：
//  1. 小程序 wx.login() 拿到 code 后调用本接口；
//  2. 后端把 code 交给 Casdoor（tag=wechat_miniprogram），Casdoor 与微信换 openid
//     并自动创建/更新 Casdoor 用户，返回 Casdoor JWT；
//  3. 校验 Casdoor JWT，取出用户 name（形如 wechat-{openid}）；
//  4. 将该用户落地到本地 user 表（首次自动注册），并签发商城自身 JWT。
//
// 本地开发（Casdoor.MockMiniProgram=true）时跳过第 2/3 步，直接用 code 模拟 openid。
func (l *MpLoginLogic) MpLogin(req *types.MpLoginRequest) (resp *types.MpLoginResponse, err error) {
	if req.Code == "" {
		return nil, errors.New("mp login code is empty")
	}

	cfg := l.svcCtx.Config.Casdoor
	var casdoorName string
	var openId string

	if cfg.MockMiniProgram {
		// 本地联调：模拟微信 openid
		openId = casdoorx.MockOpenId(req.Code)
		casdoorName = "wechat-" + openId
		l.Logger.Infof("mock mini program login, code=%s -> casdoorName=%s", req.Code, casdoorName)
	} else {
		if cfg.Endpoint == "" {
			return nil, errors.New("casdoor endpoint not configured")
		}
		// 1) 用 code 向 Casdoor 换取 access_token
		tokenResp, err := casdoorx.ExchangeMiniProgramCode(l.ctx, cfg, req.Code, req.Username, req.Avatar)
		if err != nil {
			l.Logger.Errorf("exchange mini program code error: %v", err)
			return nil, err
		}
		// 2) 校验 Casdoor JWT，得到 Casdoor 用户（name 形如 wechat-{openid}）
		claims, err := casdoorx.ParseToken(l.ctx, cfg, tokenResp.EffectiveAccessToken())
		if err != nil {
			l.Logger.Errorf("parse casdoor token error: %v", err)
			return nil, err
		}
		casdoorName = claims.Name
		openId = strings.TrimPrefix(claims.Name, "wechat-")
		l.Logger.Infof("casdoor user %s logged in via mini program", casdoorName)
	}

	// 3) 落地本地用户（首次自动创建）
	res, err := l.svcCtx.UserRpc.LoginByCasdoor(l.ctx, &user.LoginByCasdoorRequest{
		CasdoorName: casdoorName,
		OpenId:      openId,
	})
	if err != nil {
		l.Logger.Errorf("login by casdoor error: %v", err)
		return nil, err
	}

	// 4) 签发商城自身 JWT
	now := time.Now().Unix()
	accessExpire := l.svcCtx.Config.Auth.AccessExpire
	accessToken, err := jwtx.GetToken(l.svcCtx.Config.Auth.AccessSecret, now, accessExpire, res.Id)
	if err != nil {
		return nil, err
	}

	return &types.MpLoginResponse{
		AccessToken:  accessToken,
		AccessExpire: now + accessExpire,
		UserId:       res.Id,
		CasdoorName:  res.CasdoorName,
	}, nil
}

package logic

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"mall/service/user/model"
	"mall/service/user/rpc/internal/svc"
	"mall/service/user/rpc/types/user"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/status"
)

type LoginByCasdoorLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginByCasdoorLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginByCasdoorLogic {
	return &LoginByCasdoorLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// LoginByCasdoor 将 Casdoor 用户（微信小程序 openid 等第三方身份）落地为本地用户：
// 已存在则直接返回，不存在则自动创建，并返回本地用户信息。
func (l *LoginByCasdoorLogic) LoginByCasdoor(in *user.LoginByCasdoorRequest) (*user.LoginByCasdoorResponse, error) {
	if in.CasdoorName == "" {
		return nil, status.Error(100, "casdoorName不能为空")
	}

	res, err := l.svcCtx.UserModel.FindOneByCasdoorName(l.ctx, in.CasdoorName)
	if err == model.ErrNotFound {
		// 首次登录：自动注册本地用户（密码随机，禁止通过手机密码方式登录）。
		// mobile 字段存微信 openid（唯一），避免多个第三方用户因空手机号撞唯一索引。
		randomPwd := randomPassword()
		u := &model.User{
			Name:        in.CasdoorName,
			Gender:      0,
			Mobile:      in.OpenId,
			Password:    randomPwd,
			CasdoorName: in.CasdoorName,
		}
		insertRes, err := l.svcCtx.UserModel.Insert(l.ctx, u)
		if err != nil {
			l.Logger.Errorf("insert casdoor user %s error: %v", in.CasdoorName, err)
			return nil, status.Error(500, "创建用户失败: "+err.Error())
		}
		id, err := insertRes.LastInsertId()
		if err != nil {
			return nil, status.Error(500, err.Error())
		}
		res = &model.User{
			Id:          id,
			Name:        u.Name,
			Gender:      u.Gender,
			Mobile:      u.Mobile,
			CasdoorName: u.CasdoorName,
		}
	} else if err != nil {
		l.Logger.Errorf("find casdoor user %s error: %v", in.CasdoorName, err)
		return nil, status.Error(500, err.Error())
	}

	return &user.LoginByCasdoorResponse{
		Id:          res.Id,
		Name:        res.Name,
		Gender:      res.Gender,
		Mobile:      res.Mobile,
		CasdoorName: res.CasdoorName,
	}, nil
}

func randomPassword() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

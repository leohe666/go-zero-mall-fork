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
// SaaS 租户维度为 (merchant_id, casdoor_id)：同一 Casdoor 用户在不同商户下视为不同账号。
// 已存在则直接返回（必要时回填真实手机号），不存在则自动创建。
func (l *LoginByCasdoorLogic) LoginByCasdoor(in *user.LoginByCasdoorRequest) (*user.LoginByCasdoorResponse, error) {
	if in.MerchantId <= 0 {
		return nil, status.Error(100, "merchantId 不能为空")
	}
	if in.CasdoorId == "" {
		return nil, status.Error(100, "casdoorId不能为空")
	}

	res, err := l.svcCtx.UserModel.FindOneByMerchantIdCasdoorId(l.ctx, in.MerchantId, in.CasdoorId)
	if err == model.ErrNotFound {
		// 首次登录：自动注册本地用户（密码随机，禁止通过手机密码方式登录）。
		u := &model.User{
			MerchantId: in.MerchantId,
			Name:       in.CasdoorName,
			Gender:     0,
			Mobile:     in.Mobile,
			Password:   randomPassword(),
			CasdoorId:  in.CasdoorId,
		}
		insertRes, err := l.svcCtx.UserModel.Insert(l.ctx, u)
		if err != nil {
			l.Logger.Errorf("insert casdoor user %s error: %v", in.CasdoorId, err)
			return nil, status.Error(500, "创建用户失败: "+err.Error())
		}
		id, err := insertRes.LastInsertId()
		if err != nil {
			return nil, status.Error(500, err.Error())
		}
		res = &model.User{
			Id:         id,
			MerchantId: u.MerchantId,
			Name:       u.Name,
			Gender:     u.Gender,
			Mobile:     u.Mobile,
			CasdoorId:  u.CasdoorId,
		}
	} else if err != nil {
		l.Logger.Errorf("find casdoor user %s error: %v", in.CasdoorId, err)
		return nil, status.Error(500, err.Error())
	} else {
		// 已存在用户：本次携带真实手机号且与库中不同，回填 mobile
		if in.Mobile != "" && res.Mobile != in.Mobile {
			// 手机号唯一约束（租户内）：若该手机号已被同商户其他用户占用则拒绝
			if other, err := l.svcCtx.UserModel.FindOneByMerchantIdMobile(l.ctx, in.MerchantId, in.Mobile); err == nil && other.Id != res.Id {
				l.Logger.Errorf("mobile %s already bound to user %d in merchant %d", in.Mobile, other.Id, in.MerchantId)
				return nil, status.Error(409, "手机号已被其他账号绑定")
			} else if err != nil && err != model.ErrNotFound {
				l.Logger.Errorf("find by mobile %s error: %v", in.Mobile, err)
				return nil, status.Error(500, err.Error())
			}
			oldMobile := res.Mobile
			if err := l.svcCtx.UserModel.UpdateMobile(l.ctx, in.MerchantId, res.Id, oldMobile, in.Mobile); err != nil {
				l.Logger.Errorf("update mobile for user %d error: %v", res.Id, err)
				return nil, status.Error(500, "更新手机号失败: "+err.Error())
			}
			res.Mobile = in.Mobile
			l.Logger.Infof("casdoor user %s mobile backfilled", in.CasdoorId)
		}
	}

	return &user.LoginByCasdoorResponse{
		Id:          res.Id,
		Name:        res.Name,
		Gender:      res.Gender,
		Mobile:      res.Mobile,
		CasdoorId:   res.CasdoorId,
		CasdoorName: res.Name,
		MerchantId:  res.MerchantId,
	}, nil
}

func randomPassword() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
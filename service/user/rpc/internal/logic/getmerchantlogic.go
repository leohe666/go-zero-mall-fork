package logic

import (
	"context"

	"mall/common/merchantx"
	"mall/service/user/model"
	"mall/service/user/rpc/internal/svc"
	"mall/service/user/rpc/types/user"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/status"
)

type GetMerchantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMerchantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMerchantLogic {
	return &GetMerchantLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// GetMerchant 返回商户配置：Casdoor 租户段 + 微信小程序凭据。
// 微信 AppSecret 与 Casdoor clientSecret 均在 user rpc 侧用平台主密钥解密后返回，
// 网关不接触主密钥（安全边界：密钥只存在于 user rpc 与数据库）。
func (l *GetMerchantLogic) GetMerchant(in *user.GetMerchantRequest) (*user.GetMerchantResponse, error) {
	res, err := l.svcCtx.MerchantModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(100, "商户不存在")
		}
		l.Logger.Errorf("find merchant %d error: %v", in.Id, err)
		return nil, status.Error(500, err.Error())
	}
	if res.Status != 1 {
		return nil, status.Error(100, "商户已停用")
	}

	wxSecret, err := merchantx.Decrypt(l.svcCtx.MerchantMasterKey, res.WxAppSecretEnc)
	if err != nil {
		l.Logger.Errorf("decrypt merchant %d wx secret error: %v", in.Id, err)
		return nil, status.Error(500, "解密商户凭据失败")
	}

	csSecret := ""
	if res.CasdoorClientSecretEnc != "" {
		csSecret, err = merchantx.Decrypt(l.svcCtx.MerchantMasterKey, res.CasdoorClientSecretEnc)
		if err != nil {
			l.Logger.Errorf("decrypt merchant %d casdoor client secret error: %v", in.Id, err)
			return nil, status.Error(500, "解密商户凭据失败")
		}
	}

	return &user.GetMerchantResponse{
		Id:                  res.Id,
		Name:                res.Name,
		Status:              res.Status,
		CasdoorEndpoint:     res.CasdoorEndpoint,
		CasdoorClientId:     res.CasdoorClientId,
		CasdoorOrg:          res.CasdoorOrg,
		CasdoorApp:          res.CasdoorApp,
		CasdoorCertPem:      res.CasdoorCertPem.String,
		CasdoorClientSecret: csSecret,
		WxAppId:             res.WxAppId,
		WxAppSecret:         wxSecret,
	}, nil
}

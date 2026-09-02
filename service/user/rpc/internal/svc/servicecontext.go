package svc

import (
	"mall/common/merchantx"
	"mall/service/user/model"
	"mall/service/user/rpc/internal/config"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config

	UserModel     model.UserModel
	MerchantModel model.MerchantModel

	// MerchantMasterKey 平台主密钥（32 字节 hex），用于解密商户 wx_app_secret_enc
	MerchantMasterKey string
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.Mysql.DataSource)

	masterKey, err := merchantx.LoadMasterKey(c.MerchantMasterKeyFile)
	if err != nil {
		logx.Errorf("load merchant master key error: %v", err)
	}

	return &ServiceContext{
		Config:            c,
		UserModel:         model.NewUserModel(conn, c.CacheRedis),
		MerchantModel:     model.NewMerchantModel(conn, c.CacheRedis),
		MerchantMasterKey: masterKey,
	}
}
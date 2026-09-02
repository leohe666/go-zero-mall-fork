package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MerchantModel = (*customMerchantModel)(nil)

type (
	// MerchantModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMerchantModel.
	MerchantModel interface {
		merchantModel
	}

	customMerchantModel struct {
		*defaultMerchantModel
	}
)

// NewMerchantModel returns a model for the database table.
func NewMerchantModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) MerchantModel {
	return &customMerchantModel{
		defaultMerchantModel: newMerchantModel(conn, c, opts...),
	}
}

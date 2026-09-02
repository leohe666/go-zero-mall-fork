package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserModel = (*customUserModel)(nil)

type (
	// UserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserModel.
	UserModel interface {
		userModel
		// FindOneByMerchantIdCasdoorId 按 (merchant_id, casdoor_id) 查询（第三方登录关联键查找）
		FindOneByMerchantIdCasdoorId(ctx context.Context, merchantId int64, casdoorId string) (*User, error)
		// UpdateMobile 回填用户真实手机号，同时清理新旧 mobile 缓存键
		UpdateMobile(ctx context.Context, merchantId int64, id int64, oldMobile, newMobile string) error
	}

	customUserModel struct {
		*defaultUserModel
	}
)

// NewUserModel returns a model for the database table.
func NewUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn, c, opts...),
	}
}

// FindOneByMerchantIdCasdoorId 按商户 + Casdoor 用户 Id 查询本地用户（SaaS 租户隔离维度）。
// casdoor_id 为非唯一索引，命中多条时返回最早创建的一条。
func (m *customUserModel) FindOneByMerchantIdCasdoorId(ctx context.Context, merchantId int64, casdoorId string) (*User, error) {
	var resp User
	key := fmt.Sprintf("cache:user:merchant:casdoorId:%v:%v", merchantId, casdoorId)
	err := m.QueryRowIndexCtx(ctx, &resp, key, m.formatPrimary, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) (i interface{}, e error) {
		query := fmt.Sprintf("select %s from %s where `merchant_id` = ? and `casdoor_id` = ? limit 1", userRows, m.table)
		if err := conn.QueryRowCtx(ctx, &resp, query, merchantId, casdoorId); err != nil {
			return nil, err
		}
		return resp.Id, nil
	}, m.queryPrimary)
	switch err {
	case nil:
		return &resp, nil
	case sqlc.ErrNotFound:
		return nil, ErrNotFound
	default:
		return nil, err
	}
}

// UpdateMobile 仅更新 mobile 字段。旧 mobile 缓存键（如历史占位）与新 mobile 键一并清理，
// 避免回填后 FindOneByMerchantIdMobile(旧值) 命中过期缓存。
func (m *customUserModel) UpdateMobile(ctx context.Context, merchantId int64, id int64, oldMobile, newMobile string) error {
	userIdKey := fmt.Sprintf("%s%v", cacheUserIdPrefix, id)
	oldMobileKey := fmt.Sprintf("%s%v:%v", cacheUserMerchantIdMobilePrefix, merchantId, oldMobile)
	newMobileKey := fmt.Sprintf("%s%v:%v", cacheUserMerchantIdMobilePrefix, merchantId, newMobile)
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		query := fmt.Sprintf("update %s set `mobile` = ? where `id` = ?", m.table)
		return conn.ExecCtx(ctx, query, newMobile, id)
	}, userIdKey, oldMobileKey, newMobileKey)
	return err
}
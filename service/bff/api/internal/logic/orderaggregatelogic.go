// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"mall/service/bff/api/internal/svc"
	"mall/service/bff/api/internal/types"
	"mall/service/order/rpc/types/order"
	"mall/service/product/rpc/types/product"
	"mall/service/user/rpc/types/user"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type OrderAggregateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewOrderAggregateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *OrderAggregateLogic {
	return &OrderAggregateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// OrderAggregate 通过订单 ID 聚合返回订单、用户、商品三部分数据。
//
// 步骤：
//  1. 先查订单，拿到 Uid / Pid；
//  2. 再并行(而非串行)调用用户、商品两个 RPC，降低整体延迟；
//  3. 把三个 RPC 的结果按前端需要的数据结构组装并返回（字段裁剪）。
func (l *OrderAggregateLogic) OrderAggregate(req *types.OrderAggregateRequest) (*types.OrderAggregateResponse, error) {
	// 1. 查订单
	orderRes, err := l.svcCtx.OrderRpc.Detail(l.ctx, &order.DetailRequest{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}

	// 2. 并行调用用户 + 商品两个 RPC
	var (
		userRes    *user.UserInfoResponse
		productRes *product.DetailResponse
	)
	if err := mr.Finish(
		func() error {
			var callErr error
			userRes, callErr = l.svcCtx.UserRpc.UserInfo(l.ctx, &user.UserInfoRequest{
				Id: orderRes.Uid,
			})
			return callErr
		},
		func() error {
			var callErr error
			productRes, callErr = l.svcCtx.ProductRpc.Detail(l.ctx, &product.DetailRequest{
				Id: orderRes.Pid,
			})
			return callErr
		},
	); err != nil {
		return nil, err
	}

	// 3. 聚合 + 裁剪：只返回前端真正需要的字段
	return &types.OrderAggregateResponse{
		Order: types.OrderDetailResponse{
			Id:     orderRes.Id,
			Uid:    orderRes.Uid,
			Pid:    orderRes.Pid,
			Amount: orderRes.Amount,
			Status: orderRes.Status,
		},
		User: types.UserInfoResponse{
			Id:     userRes.Id,
			Name:   userRes.Name,
			Gender: userRes.Gender,
			Mobile: userRes.Mobile,
		},
		Product: types.ProductDetailResponse{
			Id:     productRes.Id,
			Name:   productRes.Name,
			Desc:   productRes.Desc,
			Stock:  productRes.Stock,
			Amount: productRes.Amount,
			Status: productRes.Status,
		},
	}, nil
}

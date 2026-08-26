// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package logic

import (
	"context"

	"mall/service/gateway/api/internal/svc"
	"mall/service/gateway/api/internal/types"
	"mall/service/product/rpc/types/product"


	"github.com/zeromicro/go-zero/core/logx"
)

type ProductRemoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewProductRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ProductRemoveLogic {
	return &ProductRemoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ProductRemoveLogic) ProductRemove(req *types.ProductRemoveRequest) (resp *types.ProductRemoveResponse, err error) {
	_, err = l.svcCtx.ProductRpc.Remove(l.ctx, &product.RemoveRequest{
		Id: req.Id,
	})
	if err != nil {
		return nil, err
	}
	return &types.ProductRemoveResponse{}, nil
}

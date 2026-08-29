package svc

import (
	"mall/service/bff/api/internal/config"
	"mall/service/order/rpc/order"
	"mall/service/product/rpc/product"
	"mall/service/user/rpc/user"

	"github.com/zeromicro/go-zero/zrpc"
)

// ServiceContext 保存下游 RPC 客户端，供 handler/logic 做多 RPC 编排与聚合。
type ServiceContext struct {
	Config     config.Config
	UserRpc    user.User
	OrderRpc   order.Order
	ProductRpc product.Product
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:     c,
		UserRpc:    user.NewUser(zrpc.MustNewClient(c.UserRpc)),
		OrderRpc:   order.NewOrder(zrpc.MustNewClient(c.OrderRpc)),
		ProductRpc: product.NewProduct(zrpc.MustNewClient(c.ProductRpc)),
	}
}

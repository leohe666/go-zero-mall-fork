// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"mall/service/gateway/api/internal/config"
	
	"github.com/zeromicro/go-zero/zrpc"
	"mall/service/order/rpc/order"
	"mall/service/pay/rpc/pay"
	"mall/service/product/rpc/product"
	"mall/service/user/rpc/user"
)

type ServiceContext struct {
	Config     config.Config
	UserRpc    user.User
	OrderRpc   order.Order
	ProductRpc product.Product
	PayRpc     pay.Pay
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:     c,
		UserRpc:    user.NewUser(zrpc.MustNewClient(c.UserRpc)),
		OrderRpc:   order.NewOrder(zrpc.MustNewClient(c.OrderRpc)),
		ProductRpc: product.NewProduct(zrpc.MustNewClient(c.ProductRpc)),
		PayRpc:     pay.NewPay(zrpc.MustNewClient(c.PayRpc)),
	}
}

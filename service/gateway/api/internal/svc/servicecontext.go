package svc

import (
	"os"

	"mall/service/gateway/api/internal/config"
	"mall/service/order/rpc/order"
	"mall/service/product/rpc/product"
	"mall/service/user/rpc/user"

	"github.com/zeromicro/go-zero/core/logx"
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
	// 读取 Casdoor 证书公钥（用于校验 Casdoor 签发的 JWT）
	if c.Casdoor.Certificate != "" {
		cert, err := os.ReadFile(c.Casdoor.Certificate)
		if err != nil {
			logx.Errorf("read casdoor certificate error: %v, path: %s", err, c.Casdoor.Certificate)
		} else {
			c.Casdoor.Certificate = string(cert)
		}
	}

	return &ServiceContext{
		Config:     c,
		UserRpc:    user.NewUser(zrpc.MustNewClient(c.UserRpc)),
		OrderRpc:   order.NewOrder(zrpc.MustNewClient(c.OrderRpc)),
		ProductRpc: product.NewProduct(zrpc.MustNewClient(c.ProductRpc)),
	}
}

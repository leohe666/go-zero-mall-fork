package config

import (
	"mall/common/casdoorx"

	"github.com/zeromicro/go-zero/gateway"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config 是统一网关的配置。
//
// 统一网关：一个进程、一个端口 (8888)。
//   - Upstreams（GatewayConf）：HTTP -> gRPC 纯透传（register / 订单 / 商品 / 支付 CRUD）
//   - Auth + 下游 RPC 客户端：login（签发 JWT）、userinfo / aggregate（校验 JWT、多 RPC 编排聚合）
//
// BFF 聚合层已合并回本网关，不再需要独立的 BFF 服务。
type Config struct {
	gateway.GatewayConf

	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	UserRpc    zrpc.RpcClientConf
	OrderRpc   zrpc.RpcClientConf
	ProductRpc zrpc.RpcClientConf

	// Casdoor SaaS 身份认证（微信小程序登录）
	Casdoor casdoorx.Config
}

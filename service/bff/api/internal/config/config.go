package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config 是 BFF 服务的配置。
//
// BFF (Backend For Frontend) 独立进程，负责需要业务编排的接口：
//   - login：签发 JWT
//   - userinfo / aggregate：校验 JWT、取 uid、多 RPC 编排聚合
//
// 下游 RPC 客户端配置（UserRpc/OrderRpc/ProductRpc）供 ServiceContext 使用。
type Config struct {
	rest.RestConf

	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	UserRpc    zrpc.RpcClientConf
	OrderRpc   zrpc.RpcClientConf
	ProductRpc zrpc.RpcClientConf
}

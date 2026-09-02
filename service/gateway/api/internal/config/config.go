package config

import (
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
//
// SaaS 多商户：每商户的 Casdoor 应用/组织 + 微信小程序凭据都存在 merchant 表（MySQL），
// 经 user rpc GetMerchant 读取（微信 AppSecret 在 user rpc 内用平台主密钥解密）。
// 网关配置文件不再持有任何商户凭据（无 Casdoor clientId/证书、无微信 AppID/Secret）。
type Config struct {
	gateway.GatewayConf

	Auth struct {
		AccessSecret string
		AccessExpire int64
	}

	UserRpc    zrpc.RpcClientConf
	OrderRpc   zrpc.RpcClientConf
	ProductRpc zrpc.RpcClientConf
}

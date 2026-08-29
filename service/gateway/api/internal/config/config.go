package config

import (
	"github.com/zeromicro/go-zero/gateway"
)

// Config 是纯透传网关的配置。
//
// 方案 B：网关只做 HTTP -> gRPC 透传（由 GatewayConf.Upstreams 驱动，零业务代码），
// 不再承载 JWT 签发与聚合编排；那些接口已拆到独立 BFF 服务 service/bff/api。
type Config struct {
	gateway.GatewayConf
}

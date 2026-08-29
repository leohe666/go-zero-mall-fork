// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"

	"mall/service/gateway/api/internal/config"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/gateway"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 纯透传网关：只根据 Upstreams 把 HTTP 直接转发到下游 gRPC，零业务代码。
	// 聚合/编排接口（login/userinfo/aggregate）已拆到独立 BFF 服务 service/bff/api（端口 8887）。
	gw := gateway.MustNewServer(c.GatewayConf)
	defer gw.Stop()

	fmt.Printf("Starting gateway at %s:%d...\n", c.Host, c.Port)
	gw.Start()
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"

	"mall/service/gateway/api/internal/config"
	"mall/service/gateway/api/internal/handler"
	"mall/service/gateway/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/gateway"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 统一网关：一个进程、一个端口 (8888)，所有业务 API 的唯一入口。
	//   - Upstreams：HTTP -> gRPC 纯透传（register / 订单 / 商品 / 支付 CRUD）
	//   - AddRoutes：login（签发 JWT）、userinfo / aggregate（校验 JWT、多 RPC 聚合编排）
	// BFF 聚合层已合并回本网关，不再需要独立的 BFF 进程/端口。
	gw := gateway.MustNewServer(c.GatewayConf)
	defer gw.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(gw.Server, ctx)

	fmt.Printf("Starting unified gateway at %s:%d...\n", c.Host, c.Port)
	gw.Start()
}

// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"

	"mall/common/response"
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
	// 开启 conf.UseEnv：允许 yaml 中 ${ENV_VAR} 从环境变量展开（如 ${WECHAT_APP_SECRET}）
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	// 统一网关：一个进程、一个端口 (8888)，所有业务 API 的唯一入口。
	//   - Upstreams：HTTP -> gRPC 纯透传（register / 订单 / 商品 / 支付 CRUD）
	//   - AddRoutes：login（签发 JWT）、userinfo / aggregate（校验 JWT、多 RPC 聚合编排）
	//   - response.Wrapper：全局中间件（rest.Server.Use 对所有路由生效），
	//     所有响应统一为 {code, msg, data} 结构
	gw := gateway.MustNewServer(c.GatewayConf)
	defer gw.Stop()

	ctx := svc.NewServiceContext(c)
	gw.Server.Use(response.Wrapper)
	handler.RegisterHandlers(gw.Server, ctx)

	fmt.Printf("Starting unified gateway at %s:%d...\n", c.Host, c.Port)
	gw.Start()
}

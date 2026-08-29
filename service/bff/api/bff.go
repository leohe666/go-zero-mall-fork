package main

import (
	"flag"
	"fmt"

	"mall/service/bff/api/internal/config"
	"mall/service/bff/api/internal/handler"
	"mall/service/bff/api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/prometheus"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/bff.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	prometheus.StartAgent(c.Prometheus)

	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting bff at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

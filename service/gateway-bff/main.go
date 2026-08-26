// Code scaffolded for BFF API Gateway.
package main

import (
	"flag"
	"fmt"

	"mall/service/gateway-bff/internal/config"
	"mall/service/gateway-bff/internal/proxy"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/gateway.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	// Register proxy routes
	for _, route := range c.Routes {
		handler, err := proxy.NewProxyHandler(route.Prefix, route.Target, route.Strip)
		if err != nil {
			panic(fmt.Sprintf("failed to create proxy for %s: %v", route.Prefix, err))
		}

		// Use empty method to match all HTTP methods
		server.AddRoute(rest.Route{
			Method:  "",
			Path:    route.Prefix + "/*",
			Handler: handler.ServeHTTP,
		})
	}

	fmt.Printf("Starting BFF Gateway at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
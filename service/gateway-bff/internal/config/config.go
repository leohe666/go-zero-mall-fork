// Code scaffolded for BFF API Gateway.
package config

import (
	"github.com/zeromicro/go-zero/rest"
)

type Route struct {
	Prefix string `json:"prefix"`
	Target string `json:"target"`
	Strip  bool   `json:"strip"`
}

type Config struct {
	rest.RestConf
	Routes []Route `json:",optional"`
}
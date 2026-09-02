package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mysql struct {
		DataSource string
	}

	CacheRedis cache.CacheConf

	Salt string

	// MerchantMasterKeyFile 平台主密钥文件路径（32 字节 hex）。
	// 生产建议 KMS 注入；开发用 gitignore 的 key 文件，禁止提交。
	MerchantMasterKeyFile string `json:"MerchantMasterKeyFile"`
}

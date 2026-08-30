# Makefile - go-zero 微服务开发命令
# 用法: make run        # 启动 9 个 air (每服务独立) + start-services.sh 进程守护
#      make build       # 仅编译 9 个服务到 tmp/
#      make air         # 仅跑 9 个 air 监听编译 (后台)
#      make services    # 仅跑 start-services.sh 进程守护
#      make logs        # 查看指定服务的 air 输出: make logs svc=user_api
#      make stop        # 停止所有后台进程
#      make clean       # 清理编译产物

# 自动检测工作目录: 容器里是 /usr/src/code, 本地是当前目录
WORKDIR := $(shell pwd)
ifeq ($(wildcard /usr/src/code/go.mod),)
  CODE_DIR := $(WORKDIR)
else
  CODE_DIR := /usr/src/code
endif

# 9 个服务对应的 air 配置 (8 个业务服务 + 1 个统一网关)
AIR_CONFIGS := .air.user_api.toml .air.product_api.toml .air.order_api.toml .air.pay_api.toml \
               .air.user_rpc.toml .air.product_rpc.toml .air.order_rpc.toml .air.pay_rpc.toml \
               .air.gateway_api.toml

# 统一网关配置
AIR_GATEWAY_API := .air.gateway_api.toml

.PHONY: run build air services logs stop clean help gateway gateway-build gateway-air

# 默认目标
help:
	@echo "可用命令:"
	@echo "  make run       - 启动开发环境 (9个 air 独立盯每服务 + 进程守护)"
	@echo "  make build     - 仅编译 9 个服务到 tmp/"
	@echo "  make air       - 仅运行 9 个 air 监听文件变化并编译 (后台)"
	@echo "  make services  - 仅运行进程守护 (需 air 在后台跑)"
	@echo "  make logs      - 查看指定服务 air 输出: make logs svc=user_api"
	@echo "  make stop      - 停止所有后台进程"
	@echo "  make clean     - 清理 tmp/ 目录"
	@echo ""
	@echo "统一网关命令:"
	@echo "  make gateway       - 单独启动统一网关 (air 热加载 + 运行)"
	@echo "  make gateway-build - 仅编译统一网关"
	@echo "  make gateway-air   - 仅运行统一网关 air 监听 (后台)"

# 启动完整开发环境: 9 个 air (后台, 输出各自日志) + start-services.sh (前台)
run:
	@echo "=== Starting dev environment with per-service hot-reload ==="
	@cd $(CODE_DIR) && \
	mkdir -p tmp && \
	for cfg in $(AIR_CONFIGS); do \
	  name=$${cfg#.air.}; name=$${name%.toml}; \
	  air -c $$cfg > tmp/air.$$name.log 2>&1 & \
	  echo "air $$cfg started (pid $$!) -> tmp/air.$$name.log"; \
	done; \
	trap "pkill -f 'air -c .air' 2>/dev/null; exit" INT TERM; \
	./start-services.sh

# 仅编译 9 个服务
build:
	@echo "=== Building 9 services to tmp/ ==="
	@cd $(CODE_DIR) && \
	go build -o ./tmp/user_api ./service/user/api && \
	go build -o ./tmp/product_api ./service/product/api && \
	go build -o ./tmp/order_api ./service/order/api && \
	go build -o ./tmp/pay_api ./service/pay/api && \
	go build -o ./tmp/user_rpc ./service/user/rpc && \
	go build -o ./tmp/product_rpc ./service/product/rpc && \
	go build -o ./tmp/order_rpc ./service/order/rpc && \
	go build -o ./tmp/pay_rpc ./service/pay/rpc && \
	go build -o ./tmp/gateway_api ./service/gateway/api && \
	echo "Build done: 9 binaries in tmp/"

# 仅跑 9 个 air (后台编译触发器)
air:
	@echo "=== Running 9 air file watchers (per-service) ==="
	@cd $(CODE_DIR) && \
	mkdir -p tmp && \
	for cfg in $(AIR_CONFIGS); do \
	  name=$${cfg#.air.}; name=$${name%.toml}; \
	  air -c $$cfg > tmp/air.$$name.log 2>&1 & \
	  echo "air $$cfg started (pid $$!) -> tmp/air.$$name.log"; \
	done; \
	echo "All 9 air watchers running in background"; \
	echo "Run 'make services' in another terminal to start services"

# 查看指定服务的 air 输出 (实时): make logs svc=user_api
logs:
	@cd $(CODE_DIR) && if [ -n "$(svc)" ]; then \
	  tail -f tmp/air.$(svc).log; \
	else \
	  echo "用法: make logs svc=<服务名>  如: make logs svc=user_api"; \
	  echo "可用的 air 日志:"; \
	  ls -1 tmp/air.*.log 2>/dev/null || echo "  (暂无, 先运行 make run)"; \
	fi

# 仅跑进程守护 (需 air 在别处跑)
services:
	@echo "=== Running service supervisor ==="
	@cd $(CODE_DIR) && ./start-services.sh

# 停止所有相关进程
stop:
	@echo "=== Stopping all dev processes ==="
	@-pkill -f "air -c .air" 2>/dev/null || true
	@-pkill -f "tmp/.*_(api|rpc)" 2>/dev/null || true
	@-pkill -f "start-services.sh" 2>/dev/null || true
	@echo "Stopped"

# 清理编译产物
clean:
	@echo "=== Cleaning tmp/ ==="
	@cd $(CODE_DIR) && rm -rf tmp/ build-errors.log
	@echo "Cleaned"

# 统一网关: 启动完整开发环境 (air + 运行)
gateway:
	@echo "=== Starting unified gateway with hot-reload ==="
	@cd $(CODE_DIR) && \
	mkdir -p tmp && \
	air -c $(AIR_GATEWAY_API) > tmp/air.gateway_api.log 2>&1 & \
	echo "air $(AIR_GATEWAY_API) started (pid $$!) -> tmp/air.gateway_api.log"; \
	trap "pkill -f 'air -c .air.gateway_api' 2>/dev/null; exit" INT TERM; \
	./tmp/gateway_api -f ./service/gateway/api/etc/gateway.yaml

# 仅编译统一网关
gateway-build:
	@echo "=== Building unified gateway to tmp/ ==="
	@cd $(CODE_DIR) && \
	go build -o ./tmp/gateway_api ./service/gateway/api && \
	echo "Build done: gateway_api in tmp/"

# 仅跑统一网关 air (后台编译触发器)
gateway-air:
	@echo "=== Running unified gateway air file watcher ==="
	@cd $(CODE_DIR) && \
	mkdir -p tmp && \
	air -c $(AIR_GATEWAY_API) > tmp/air.gateway_api.log 2>&1 & \
	echo "air $(AIR_GATEWAY_API) started (pid $$!) -> tmp/air.gateway_api.log"; \
	echo "Gateway air watcher running in background"; \
	echo "Run './tmp/gateway_api -f ./service/gateway/api/etc/gateway.yaml' to start"

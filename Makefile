# Makefile - go-zero 微服务开发命令
# 用法: make run        # 启动 8 个 air (每服务独立) + start-services.sh 进程守护
#      make build       # 仅编译 8 个服务到 tmp/
#      make air         # 仅跑 8 个 air 监听编译 (后台)
#      make services    # 仅跑 start-services.sh 进程守护
#      make stop        # 停止所有后台进程
#      make clean       # 清理编译产物

# 自动检测工作目录: 容器里是 /usr/src/code, 本地是当前目录
WORKDIR := $(shell pwd)
ifeq ($(wildcard /usr/src/code/go.mod),)
  CODE_DIR := $(WORKDIR)
else
  CODE_DIR := /usr/src/code
endif

# 8 个服务对应的 air 配置
AIR_CONFIGS := .air.user_api.toml .air.product_api.toml .air.order_api.toml .air.pay_api.toml \
               .air.user_rpc.toml .air.product_rpc.toml .air.order_rpc.toml .air.pay_rpc.toml

.PHONY: run build air services stop clean help

# 默认目标
help:
	@echo "可用命令:"
	@echo "  make run       - 启动开发环境 (8个 air 独立盯每服务 + 进程守护)"
	@echo "  make build     - 仅编译 8 个服务到 tmp/"
	@echo "  make air       - 仅运行 8 个 air 监听文件变化并编译 (后台)"
	@echo "  make services  - 仅运行进程守护 (需 air 在后台跑)"
	@echo "  make stop      - 停止所有后台进程"
	@echo "  make clean     - 清理 tmp/ 目录"

# 启动完整开发环境: 8 个 air (后台) + start-services.sh (前台)
run:
	@echo "=== Starting dev environment with per-service hot-reload ==="
	@cd $(CODE_DIR) && \
	for cfg in $(AIR_CONFIGS); do \
	  air -c $$cfg & \
	  echo "air $$cfg started (pid $$!)"; \
	done; \
	AIR_PIDS=$$!; \
	trap "pkill -f 'air -c .air' 2>/dev/null; exit" INT TERM; \
	./start-services.sh

# 仅编译 8 个服务
build:
	@echo "=== Building 8 services to tmp/ ==="
	@cd $(CODE_DIR) && \
	go build -o ./tmp/user_api ./service/user/api && \
	go build -o ./tmp/product_api ./service/product/api && \
	go build -o ./tmp/order_api ./service/order/api && \
	go build -o ./tmp/pay_api ./service/pay/api && \
	go build -o ./tmp/user_rpc ./service/user/rpc && \
	go build -o ./tmp/product_rpc ./service/product/rpc && \
	go build -o ./tmp/order_rpc ./service/order/rpc && \
	go build -o ./tmp/pay_rpc ./service/pay/rpc && \
	echo "Build done: 8 binaries in tmp/"

# 仅跑 8 个 air (后台编译触发器)
air:
	@echo "=== Running 8 air file watchers (per-service) ==="
	@cd $(CODE_DIR) && \
	for cfg in $(AIR_CONFIGS); do \
	  air -c $$cfg & \
	  echo "air $$cfg started (pid $$!)"; \
	done; \
	echo "All 8 air watchers running in background"; \
	echo "Run 'make services' in another terminal to start services"

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
# Makefile - go-zero 微服务开发命令
# 用法: make run        # 启动所有服务 + 热加载 (air + start-services.sh)
#      make build       # 仅编译 8 个服务到 tmp/
#      make air         # 仅跑 air 监听编译
#      make services    # 仅跑 start-services.sh 进程守护
#      make stop        # 停止所有后台进程
#      make clean       # 清理编译产物

.PHONY: run build air services stop clean help

# 默认目标
help:
	@echo "可用命令:"
	@echo "  make run       - 启动开发环境 (air 热编译 + 8服务热重启)"
	@echo "  make build     - 仅编译 8 个服务到 tmp/"
	@echo "  make air       - 仅运行 air 监听文件变化并编译"
	@echo "  make services  - 仅运行进程守护 (需 air 在后台跑)"
	@echo "  make stop      - 停止所有后台进程"
	@echo "  make clean     - 清理 tmp/ 目录"

# 启动完整开发环境: air (后台) + start-services.sh (前台)
run:
	@echo "=== Starting dev environment with hot-reload ==="
	@cd /usr/src/code && air -c .air.toml & \
	AIR_PID=$$!; \
	echo "air started (pid $$AIR_PID)"; \
	trap "kill $$AIR_PID 2>/dev/null; exit" INT TERM; \
	./start-services.sh

# 仅编译 8 个服务
build:
	@echo "=== Building 8 services to tmp/ ==="
	@cd /usr/src/code && \
	go build -o ./tmp/user_api ./service/user/api && \
	go build -o ./tmp/product_api ./service/product/api && \
	go build -o ./tmp/order_api ./service/order/api && \
	go build -o ./tmp/pay_api ./service/pay/api && \
	go build -o ./tmp/user_rpc ./service/user/rpc && \
	go build -o ./tmp/product_rpc ./service/product/rpc && \
	go build -o ./tmp/order_rpc ./service/order/rpc && \
	go build -o ./tmp/pay_rpc ./service/pay/rpc && \
	echo "Build done: 8 binaries in tmp/"

# 仅跑 air (后台编译触发器)
air:
	@echo "=== Running air file watcher ==="
	@cd /usr/src/code && air -c .air.toml

# 仅跑进程守护 (需 air 在别处跑)
services:
	@echo "=== Running service supervisor ==="
	@cd /usr/src/code && ./start-services.sh

# 停止所有相关进程
stop:
	@echo "=== Stopping all dev processes ==="
	@-pkill -f "air -c .air.toml" 2>/dev/null || true
	@-pkill -f "tmp/.*_(api|rpc)" 2>/dev/null || true
	@-pkill -f "start-services.sh" 2>/dev/null || true
	@echo "Stopped"

# 清理编译产物
clean:
	@echo "=== Cleaning tmp/ ==="
	@cd /usr/src/code && rm -rf tmp/ build-errors.log
	@echo "Cleaned"
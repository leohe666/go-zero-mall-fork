#!/bin/bash
# start-services.sh - 启动所有 API 服务 + RPC 服务，配合每服务独立的 air 热加载
# air (.air.<service>.toml) 只监听对应服务目录并编译到 tmp/，
# 本脚本监控 tmp/ 下二进制文件变化，只重启发生变化的那个服务
# 兼容 Linux(容器 /usr/src/code) 和 macOS(本地脚本目录)

set -e

# 自动检测工作目录: 容器里是 /usr/src/code, 本地是脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "/usr/src/code/go.mod" ]; then
    CODE_DIR="/usr/src/code"
else
    CODE_DIR="$SCRIPT_DIR"
fi

cd "$CODE_DIR"

echo "=== Starting all services with air hot-reload ==="

# 先编译一次
echo "Initial build..."
go build -o ./tmp/user_api ./service/user/api
go build -o ./tmp/product_api ./service/product/api
go build -o ./tmp/order_api ./service/order/api
go build -o ./tmp/pay_api ./service/pay/api
go build -o ./tmp/user_rpc ./service/user/rpc
go build -o ./tmp/product_rpc ./service/product/rpc
go build -o ./tmp/order_rpc ./service/order/rpc
go build -o ./tmp/pay_rpc ./service/pay/rpc
go build -o ./tmp/gateway_api ./service/gateway/api

# 二进制签名 = "mtime(秒):size"，同时兼容 GNU stat(Linux) 与 BSD stat(macOS)
# 相比只比较 mtime，加 size 可避免同秒内两次构建(秒级 mtime 相同)漏检
get_mtime() {
    stat -c %Y "$1" 2>/dev/null || stat -f %m "$1" 2>/dev/null || echo 0
}

get_size() {
    stat -c %s "$1" 2>/dev/null || stat -f %z "$1" 2>/dev/null || echo 0
}

get_sig() {
    echo "$(get_mtime "$1"):$(get_size "$1")"
}

USER_API_SIG=$(get_sig ./tmp/user_api)
PRODUCT_API_SIG=$(get_sig ./tmp/product_api)
ORDER_API_SIG=$(get_sig ./tmp/order_api)
PAY_API_SIG=$(get_sig ./tmp/pay_api)
USER_RPC_SIG=$(get_sig ./tmp/user_rpc)
PRODUCT_RPC_SIG=$(get_sig ./tmp/product_rpc)
ORDER_RPC_SIG=$(get_sig ./tmp/order_rpc)
PAY_RPC_SIG=$(get_sig ./tmp/pay_rpc)
GATEWAY_API_SIG=$(get_sig ./tmp/gateway_api)

# 等待端口释放
wait_port_free() {
    local port=$1
    local timeout=15
    local start=$(date +%s)
    # 使用多种方式检查端口 (lsof 同时兼容 macOS/Linux，优先)
    while true; do
        local PORT_IN_USE=0
        if command -v lsof >/dev/null 2>&1; then
            lsof -i :$port 2>/dev/null | grep -q LISTEN && PORT_IN_USE=1
        elif command -v ss >/dev/null 2>&1; then
            ss -tlnp 2>/dev/null | grep -q ":$port " && PORT_IN_USE=1
            # 也检查 TIME_WAIT
            ss -tanp 2>/dev/null | grep -q ":$port " && PORT_IN_USE=1
        elif command -v netstat >/dev/null 2>&1; then
            netstat -tlnp 2>/dev/null | grep -q ":$port " && PORT_IN_USE=1
            netstat -tanp 2>/dev/null | grep -q ":$port " && PORT_IN_USE=1
        else
            (timeout 1 bash -c "cat < /dev/null > /dev/tcp/127.0.0.1/$port" 2>/dev/null) && PORT_IN_USE=1
        fi
        if [ $PORT_IN_USE -eq 0 ]; then
            # 额外等待 0.5s 确保内核释放
            sleep 0.5
            return 0
        fi
        if [ $(($(date +%s) - start)) -gt $timeout ]; then
            echo "Warning: port $port still in use after ${timeout}s, trying kill -9..."
            if command -v lsof >/dev/null 2>&1; then
                lsof -ti :$port | xargs -r kill -9 2>/dev/null || true
            elif command -v ss >/dev/null 2>&1; then
                ss -tlnp 2>/dev/null | grep ":$port " | grep -o 'pid=[0-9]*' | cut -d= -f2 | xargs -r kill -9 2>/dev/null || true
            fi
            sleep 1
            return 0
        fi
        sleep 0.3
    done
}

# 启动服务的函数
start_user_rpc() { ./tmp/user_rpc -f ./service/user/rpc/etc/user.yaml & }
start_product_rpc() { ./tmp/product_rpc -f ./service/product/rpc/etc/product.yaml & }
start_order_rpc() { ./tmp/order_rpc -f ./service/order/rpc/etc/order.yaml & }
start_pay_rpc() { ./tmp/pay_rpc -f ./service/pay/rpc/etc/pay.yaml & }
start_user_api() { ./tmp/user_api -f ./service/user/api/etc/user.yaml & }
start_product_api() { ./tmp/product_api -f ./service/product/api/etc/product.yaml & }
start_order_api() { ./tmp/order_api -f ./service/order/api/etc/order.yaml & }
start_pay_api() { ./tmp/pay_api -f ./service/pay/api/etc/pay.yaml & }
start_gateway_api() { ./tmp/gateway_api -f ./service/gateway/api/etc/gateway.yaml & }

# 每服务"重启中"标记 (bash3 兼容写法)，避免同一服务在重启期间被再次触发导致并发重启竞态
restart_busy() {
    local flag
    eval "flag=\${RESTARTING_$1:-0}"
    [ "$flag" = "1" ]
}
set_busy()   { eval "RESTARTING_$1=1"; }
clear_busy() { eval "RESTARTING_$1=0"; }

# 重启单个服务
restart_service() {
    local name=$1
    local pid_var=$2
    local sig_var=$3
    local start_func=$4
    local port=$5

    set_busy "$name"
    echo "[hot-reload] $name binary updated, restarting..."
    local old_pid=${!pid_var}
    kill $old_pid 2>/dev/null || true
    # 1) 等进程彻底退出
    local start_wait=$(date +%s)
    while kill -0 $old_pid 2>/dev/null; do
        if [ $(($(date +%s) - start_wait)) -gt 10 ]; then
            echo "Warning: $name (pid $old_pid) still alive after 10s, SIGKILL..."
            kill -9 $old_pid 2>/dev/null || true
            sleep 0.5
            break
        fi
        sleep 0.2
    done
    # 2) 激进清理端口
    wait_port_free $port
    # 3) 等二进制写完整且是合法可执行文件（避免 air 写到一半就被检测到）
    local bin_path="./tmp/$name"
    local max_wait=15
    local start_bin=$(date +%s)
    while true; do
        # 检查文件大小稳定
        local curr_size=$(get_size "$bin_path")
        if [ "$curr_size" -gt 0 ]; then
            # ELF(Linux/容器) / Mach-O(macOS 本地) 均为合法可执行文件
            if file "$bin_path" 2>/dev/null | grep -q "executable"; then
                echo "[hot-reload] $name binary ready (size: $curr_size)"
                break
            fi
        fi
        if [ $(($(date +%s) - start_bin)) -gt $max_wait ]; then
            echo "Warning: $name binary not ready after ${max_wait}s, retrying anyway..."
            break
        fi
        sleep 0.3
    done
    # 4) 额外等待 1s 确保内核完全释放
    sleep 1
    # 5) 启动新进程
    $start_func
    eval "$pid_var=\\$!"
    eval "$sig_var=\\$(get_sig ./tmp/$name)"
    echo "[hot-reload] $name restarted (new pid ${!pid_var})"
    clear_busy "$name"
}

# 启动所有服务
start_user_rpc; USER_RPC_PID=$!
start_product_rpc; PRODUCT_RPC_PID=$!
start_order_rpc; ORDER_RPC_PID=$!
start_pay_rpc; PAY_RPC_PID=$!
start_user_api; USER_API_PID=$!
start_product_api; PRODUCT_API_PID=$!
start_order_api; ORDER_API_PID=$!
start_pay_api; PAY_API_PID=$!
start_gateway_api; GATEWAY_API_PID=$!

echo "Started all 9 services (4 RPC + 4 API + 1 BFF Gateway)"

# 监控循环：每 2 秒检查二进制签名，只重启签名(内容)发生变化的服务
while true; do
    sleep 2

    for name in user_rpc product_rpc order_rpc pay_rpc user_api product_api order_api pay_api gateway_api; do
        # 该服务正在重启中则跳过，等重启完成后再检测，避免并发重启同一服务
        restart_busy "$name" && continue

        NAME_UP=$(echo "$name" | tr '[:lower:]' '[:upper:]')
        NAME_UP=$(echo "$NAME_UP" | tr '-' '_')
        eval "CURR_SIG=\$${NAME_UP}_SIG"
        NEW_SIG=$(get_sig ./tmp/$name)

        if [ "$NEW_SIG" != "$CURR_SIG" ]; then
            case $name in
                user_rpc) restart_service "user_rpc" USER_RPC_PID USER_RPC_SIG start_user_rpc 9000 ;;
                product_rpc) restart_service "product_rpc" PRODUCT_RPC_PID PRODUCT_RPC_SIG start_product_rpc 9001 ;;
                order_rpc) restart_service "order_rpc" ORDER_RPC_PID ORDER_RPC_SIG start_order_rpc 9002 ;;
                pay_rpc) restart_service "pay_rpc" PAY_RPC_PID PAY_RPC_SIG start_pay_rpc 9003 ;;
                user_api) restart_service "user_api" USER_API_PID USER_API_SIG start_user_api 8000 ;;
                product_api) restart_service "product_api" PRODUCT_API_PID PRODUCT_API_SIG start_product_api 8001 ;;
                order_api) restart_service "order_api" ORDER_API_PID ORDER_API_SIG start_order_api 8002 ;;
                pay_api) restart_service "pay_api" PAY_API_PID PAY_API_SIG start_pay_api 8003 ;;
                gateway_api) restart_service "gateway_api" GATEWAY_API_PID GATEWAY_API_SIG start_gateway_api 8888 ;;
            esac
        fi
    done
done

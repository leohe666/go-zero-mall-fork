#!/bin/bash
# start-services.sh - 启动所有 API 服务 + RPC 服务，配合 air 热加载
# 监控 tmp/ 下的二进制文件变化，自动重启对应服务

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

# 记录二进制文件的修改时间
get_mtime() {
    stat -c %Y "$1" 2>/dev/null || echo 0
}

USER_API_MTIME=$(get_mtime ./tmp/user_api)
PRODUCT_API_MTIME=$(get_mtime ./tmp/product_api)
ORDER_API_MTIME=$(get_mtime ./tmp/order_api)
PAY_API_MTIME=$(get_mtime ./tmp/pay_api)
USER_RPC_MTIME=$(get_mtime ./tmp/user_rpc)
PRODUCT_RPC_MTIME=$(get_mtime ./tmp/product_rpc)
ORDER_RPC_MTIME=$(get_mtime ./tmp/order_rpc)
PAY_RPC_MTIME=$(get_mtime ./tmp/pay_rpc)

# 等待端口释放
wait_port_free() {
    local port=$1
    local timeout=15
    local start=$(date +%s)
    # 使用多种方式检查端口
    while true; do
        local PORT_IN_USE=0
        if command -v ss >/dev/null 2>&1; then
            ss -tlnp 2>/dev/null | grep -q ":$port " && PORT_IN_USE=1
            # 也检查 TIME_WAIT
            ss -tanp 2>/dev/null | grep -q ":$port " && PORT_IN_USE=1
        elif command -v netstat >/dev/null 2>&1; then
            netstat -tlnp 2>/dev/null | grep -q ":$port " && PORT_IN_USE=1
            netstat -tanp 2>/dev/null | grep -q ":$port " && PORT_IN_USE=1
        elif command -v lsof >/dev/null 2>&1; then
            lsof -i :$port 2>/dev/null | grep -q LISTEN && PORT_IN_USE=1
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

# 重启单个服务
restart_service() {
    local name=$1
    local pid_var=$2
    local mtime_var=$3
    local start_func=$4
    local port=$5
    
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
    # 3) 等二进制写完整且是合法 ELF（避免 air 写到一半就被检测到）
    local bin_path="./tmp/$name"
    local max_wait=15
    local start_bin=$(date +%s)
    while true; do
        # 检查文件大小稳定
        local curr_size=$(stat -c %s "$bin_path" 2>/dev/null || echo 0)
        if [ "$curr_size" -gt 0 ]; then
            # 检查是否为合法 ELF 可执行文件
            if file "$bin_path" 2>/dev/null | grep -q "ELF.*executable"; then
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
    eval "$mtime_var=\\$(get_mtime ./tmp/$name)"
    echo "[hot-reload] $name restarted (new pid ${!pid_var})"
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

echo "Started all 8 services (4 RPC + 4 API)"

# 监控循环：每 2 秒检查二进制是否更新
while true; do
    sleep 2
    
    # 检查每个服务的二进制是否有新版本
    NEW_MTIME=$(get_mtime ./tmp/user_rpc)
    if [ "$NEW_MTIME" -gt "$USER_RPC_MTIME" ]; then
        restart_service "user_rpc" USER_RPC_PID USER_RPC_MTIME start_user_rpc 9000
    fi
    
    NEW_MTIME=$(get_mtime ./tmp/product_rpc)
    if [ "$NEW_MTIME" -gt "$PRODUCT_RPC_MTIME" ]; then
        restart_service "product_rpc" PRODUCT_RPC_PID PRODUCT_RPC_MTIME start_product_rpc 9001
    fi
    
    NEW_MTIME=$(get_mtime ./tmp/order_rpc)
    if [ "$NEW_MTIME" -gt "$ORDER_RPC_MTIME" ]; then
        restart_service "order_rpc" ORDER_RPC_PID ORDER_RPC_MTIME start_order_rpc 9002
    fi
    
    NEW_MTIME=$(get_mtime ./tmp/pay_rpc)
    if [ "$NEW_MTIME" -gt "$PAY_RPC_MTIME" ]; then
        restart_service "pay_rpc" PAY_RPC_PID PAY_RPC_MTIME start_pay_rpc 9003
    fi
    
    NEW_MTIME=$(get_mtime ./tmp/user_api)
    if [ "$NEW_MTIME" -gt "$USER_API_MTIME" ]; then
        restart_service "user_api" USER_API_PID USER_API_MTIME start_user_api 8000
    fi
    
    NEW_MTIME=$(get_mtime ./tmp/product_api)
    if [ "$NEW_MTIME" -gt "$PRODUCT_API_MTIME" ]; then
        restart_service "product_api" PRODUCT_API_PID PRODUCT_API_MTIME start_product_api 8001
    fi
    
    NEW_MTIME=$(get_mtime ./tmp/order_api)
    if [ "$NEW_MTIME" -gt "$ORDER_API_MTIME" ]; then
        restart_service "order_api" ORDER_API_PID ORDER_API_MTIME start_order_api 8002
    fi
    
    NEW_MTIME=$(get_mtime ./tmp/pay_api)
    if [ "$NEW_MTIME" -gt "$PAY_API_MTIME" ]; then
        restart_service "pay_api" PAY_API_PID PAY_API_MTIME start_pay_api 8003
    fi
done
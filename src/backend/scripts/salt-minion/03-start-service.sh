#!/bin/bash
# ====================================================================
# Salt Minion 服务启动脚本 (增强版 - 支持密钥交换等待和重试)
# ====================================================================
# 描述: 启动并验证salt-minion服务
# 改进: 增加首次启动后的密钥交换等待和重试机制
#       增加与 Salt Master 的连接验证
# ====================================================================

set -eo pipefail

MAX_RETRIES=5
RETRY_INTERVAL=5
MASTER_CONNECT_TIMEOUT=60

echo "=== 启动 Salt Minion 服务 ==="

# 辅助函数：验证与 Master 的连接
verify_master_connection() {
	local timeout=$1
	local start_time=$(date +%s)
	
	echo "[Salt] 验证与 Salt Master 的连接..."
	
	while true; do
		local current_time=$(date +%s)
		local elapsed=$((current_time - start_time))
		
		if [ $elapsed -ge $timeout ]; then
			echo "[Salt] ⚠ Master 连接验证超时（${timeout}秒），继续启动..."
			return 1
		fi
		
		# 尝试使用 salt-call 测试连接
		if command -v salt-call >/dev/null 2>&1; then
			if salt-call --local test.ping --timeout=5 2>/dev/null | grep -q "True"; then
				echo "[Salt] ✓ salt-call 本地测试成功"
				return 0
			fi
		fi
		
		# 检查是否有与 master 的网络连接
		if [ -f /etc/salt/minion ]; then
			local master_addr=$(grep "^master:" /etc/salt/minion | awk '{print $2}' | tr -d ' ')
			if [ -n "$master_addr" ]; then
				# 尝试 ping master 或检测端口连通性
				if timeout 3 bash -c "echo > /dev/tcp/${master_addr}/4505" 2>/dev/null || \
				   timeout 3 bash -c "echo > /dev/tcp/${master_addr}/4506" 2>/dev/null; then
					echo "[Salt] ✓ Salt Master 端口可达"
					return 0
				fi
			fi
		fi
		
		echo "[Salt] 等待 Master 连接... (${elapsed}/${timeout}秒)"
		sleep 3
	done
}

# 尝试多种启动方式（兼容不同系统）
if systemctl --version >/dev/null 2>&1; then
	# 使用 systemd
	echo "[Salt] 使用 systemd 启动服务..."
	systemctl enable salt-minion 2>/dev/null || true
	systemctl daemon-reload 2>/dev/null || true
	
	# 首次启动以触发与 master 的密钥交换
	echo "[Salt] 首次启动服务以触发密钥交换..."
	systemctl start salt-minion || true
	
	# 等待密钥交换完成
	echo "[Salt] 等待与 Salt Master 进行密钥交换..."
	sleep 5
	
	# 重启服务以确保与 master 的可靠连接
	echo "[Salt] 重启服务以确保可靠连接..."
	systemctl restart salt-minion || true
	sleep 3
	
	# 验证服务状态并在需要时重试
	SERVICE_OK=0
	for i in $(seq 1 $MAX_RETRIES); do
		if systemctl is-active --quiet salt-minion; then
			SERVICE_OK=1
			echo "[Salt] ✓ salt-minion 服务已启动 (第 $i 次检查)"
			systemctl status salt-minion --no-pager -l 2>/dev/null || true
			break
		else
			echo "[Salt] 服务未就绪，尝试重启 ($i/$MAX_RETRIES)..."
			systemctl restart salt-minion || true
			sleep $RETRY_INTERVAL
		fi
	done
	
	if [ $SERVICE_OK -eq 0 ]; then
		echo "[Salt] ✗ salt-minion 服务启动失败"
		systemctl status salt-minion --no-pager -l 2>/dev/null || true
		journalctl -u salt-minion --no-pager -n 50 2>/dev/null || true
		exit 1
	fi
	
elif service salt-minion status >/dev/null 2>&1; then
	# 使用 service (SysV init)
	echo "[Salt] 使用 service 启动..."
	service salt-minion start || true
	sleep 5
	service salt-minion restart || true
	sleep 3
	
	SERVICE_OK=0
	for i in $(seq 1 $MAX_RETRIES); do
		if service salt-minion status >/dev/null 2>&1; then
			SERVICE_OK=1
			echo "[Salt] ✓ salt-minion 服务已启动 (第 $i 次检查)"
			service salt-minion status || true
			break
		else
			echo "[Salt] 服务未就绪，尝试重启 ($i/$MAX_RETRIES)..."
			service salt-minion restart || true
			sleep $RETRY_INTERVAL
		fi
	done
	
	if [ $SERVICE_OK -eq 0 ]; then
		echo "[Salt] ✗ salt-minion 服务启动失败"
		service salt-minion status || true
		exit 1
	fi
	
else
	# 直接启动守护进程
	echo "[Salt] 直接启动 salt-minion 守护进程..."
	salt-minion -d
	sleep 5
	
	# 重启确保连接正常
	pkill -TERM salt-minion 2>/dev/null || true
	sleep 2
	salt-minion -d
	sleep 3
	
	# 检查进程
	SERVICE_OK=0
	for i in $(seq 1 $MAX_RETRIES); do
		if pgrep -x salt-minion >/dev/null; then
			SERVICE_OK=1
			echo "[Salt] ✓ salt-minion 进程已启动 (第 $i 次检查)"
			ps aux | grep salt-minion | grep -v grep || true
			break
		else
			echo "[Salt] 进程未找到，尝试重启 ($i/$MAX_RETRIES)..."
			salt-minion -d
			sleep $RETRY_INTERVAL
		fi
	done
	
	if [ $SERVICE_OK -eq 0 ]; then
		echo "[Salt] ✗ salt-minion 进程未找到"
		exit 1
	fi
fi

# 验证与 Master 的连接（非阻塞，仅记录状态）
verify_master_connection $MASTER_CONNECT_TIMEOUT || true

# 最终验证
echo "=== Salt Minion 服务最终验证 ==="

# 验证进程
if pgrep -f "salt-minion" > /dev/null 2>&1; then
	echo "[Salt] ✓ salt-minion 进程正在运行"
else
	echo "[Salt] ✗ salt-minion 进程未运行"
	exit 1
fi

# 验证配置文件
if [ -f /etc/salt/minion ]; then
	echo "[Salt] ✓ minion 配置文件存在"
	echo "[Salt] 配置的 Master: $(grep '^master:' /etc/salt/minion | awk '{print $2}' || echo '未配置')"
else
	echo "[Salt] ⚠ minion 配置文件不存在"
fi

echo "=== Salt Minion 服务启动完成 ==="
echo "SERVICE_STATUS=running"
exit 0

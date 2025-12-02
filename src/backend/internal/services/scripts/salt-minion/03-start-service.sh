#!/bin/bash
# ====================================================================
# Salt Minion 服务启动脚本 (增强版 - 支持密钥交换等待和重试)
# ====================================================================
# 描述: 启动并验证salt-minion服务
# 改进: 增加首次启动后的密钥交换等待和重试机制
# ====================================================================

set -eo pipefail

MAX_RETRIES=3
RETRY_INTERVAL=5

echo "=== 启动 Salt Minion 服务 ==="

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
			systemctl status salt-minion --no-pager -l || true
			break
		else
			echo "[Salt] 服务未就绪，尝试重启 ($i/$MAX_RETRIES)..."
			systemctl restart salt-minion || true
			sleep $RETRY_INTERVAL
		fi
	done
	
	if [ $SERVICE_OK -eq 0 ]; then
		echo "[Salt] ✗ salt-minion 服务启动失败"
		systemctl status salt-minion --no-pager -l || true
		journalctl -u salt-minion --no-pager -n 50 || true
		exit 1
	fi
	
elif service salt-minion status >/dev/null 2>&1; then
	# 使用 service (SysV init)
	echo "[Salt] 使用 service 启动..."
	service salt-minion start || true
	sleep 5
	service salt-minion restart || true
	sleep 3
	service salt-minion status || true
	
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
	if pgrep -x salt-minion >/dev/null; then
		echo "[Salt] ✓ salt-minion 进程已启动"
		ps aux | grep salt-minion | grep -v grep || true
	else
		echo "[Salt] ✗ salt-minion 进程未找到"
		exit 1
	fi
fi

echo "=== Salt Minion 服务启动完成 ==="
exit 0

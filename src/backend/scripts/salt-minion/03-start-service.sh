#!/bin/bash
# ====================================================================
# Salt Minion 服务启动脚本
# ====================================================================
# 描述: 启动并验证salt-minion服务
# ====================================================================

set -e

echo "=== 启动 Salt Minion 服务 ==="

# 尝试多种启动方式（兼容不同系统）
if systemctl --version >/dev/null 2>&1; then
	# 使用 systemd
	echo "[Salt] 使用 systemd 启动服务..."
	systemctl enable salt-minion 2>/dev/null || true
	systemctl daemon-reload 2>/dev/null || true
	systemctl restart salt-minion
	
	# 等待服务启动
	sleep 2
	
	# 检查服务状态
	if systemctl is-active --quiet salt-minion; then
		echo "[Salt] ✓ salt-minion 服务已启动"
		systemctl status salt-minion --no-pager -l || true
	else
		echo "[Salt] ✗ salt-minion 服务启动失败"
		systemctl status salt-minion --no-pager -l || true
		journalctl -u salt-minion --no-pager -n 50 || true
		exit 1
	fi
	
elif service salt-minion status >/dev/null 2>&1; then
	# 使用 service (SysV init)
	echo "[Salt] 使用 service 启动..."
	service salt-minion start || service salt-minion restart
	sleep 2
	service salt-minion status || true
	
else
	# 直接启动守护进程
	echo "[Salt] 直接启动 salt-minion 守护进程..."
	salt-minion -d
	sleep 2
	
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

#!/bin/bash
# ====================================================================
# Salt Minion 状态验证脚本
# ====================================================================
# 描述: 验证salt-minion服务状态和日志
# ====================================================================

set -e  # 遇到错误立即退出

echo "=== 验证 Salt Minion 状态 ==="

# 检查 salt-minion 命令
if command -v salt-minion >/dev/null 2>&1; then
	echo "[Salt] ✓ salt-minion 命令可用"
	salt-minion --version || true
else
	echo "[Salt] ✗ salt-minion 命令未找到"
	exit 1
fi

# 检查配置文件
if [ -f /etc/salt/minion.d/99-master-address.conf ]; then
	echo "[Salt] ✓ Master配置文件存在"
	echo "Master配置:"
	cat /etc/salt/minion.d/99-master-address.conf
else
	echo "[Salt] ✗ Master配置文件不存在"
	exit 1
fi

# 检查服务状态
echo ""
echo "=== 服务状态 ==="
SERVICE_RUNNING=false

if systemctl --version >/dev/null 2>&1; then
	if systemctl is-active --quiet salt-minion; then
		echo "[Salt] ✓ salt-minion 服务正在运行 (systemd)"
		SERVICE_RUNNING=true
		systemctl status salt-minion --no-pager -l || true
		echo ""
		echo "=== 最近日志 ==="
		journalctl -u salt-minion --no-pager -n 20 || true
	else
		echo "[Salt] ✗ salt-minion 服务未运行 (systemd)"
		systemctl status salt-minion --no-pager -l || true
	fi
elif service salt-minion status >/dev/null 2>&1; then
	echo "[Salt] ✓ salt-minion 服务正在运行 (SysV init)"
	SERVICE_RUNNING=true
	service salt-minion status || true
else
	# 检查进程
	if pgrep -x salt-minion >/dev/null; then
		echo "[Salt] ✓ salt-minion 进程正在运行"
		SERVICE_RUNNING=true
		ps aux | grep salt-minion | grep -v grep || true
	else
		echo "[Salt] ✗ salt-minion 进程未找到"
	fi
fi

# 如果服务未运行，退出并返回错误
if [ "$SERVICE_RUNNING" = false ]; then
	echo ""
	echo "[Salt] ✗ 验证失败: salt-minion 服务未运行"
	exit 1
fi

# 检查网络连接
echo ""
echo "=== 网络连接 ==="
if command -v ss >/dev/null 2>&1; then
	ss -tuln | grep -E "(4505|4506)" || echo "未发现到Master的连接（可能正在建立）"
elif command -v netstat >/dev/null 2>&1; then
	netstat -tuln | grep -E "(4505|4506)" || echo "未发现到Master的连接（可能正在建立）"
fi

echo ""
echo "=== Salt Minion 验证成功 ==="
exit 0

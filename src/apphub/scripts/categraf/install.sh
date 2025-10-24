#!/bin/bash
set -e

echo "Installing Categraf..."

# 创建安装目录
mkdir -p /usr/local/categraf/{bin,conf,logs}

# 复制文件
cp -r bin/* /usr/local/categraf/bin/
cp -r conf/* /usr/local/categraf/conf/
chmod +x /usr/local/categraf/bin/categraf

# 安装 systemd 服务
if [ -d /etc/systemd/system ]; then
    cp categraf.service /etc/systemd/system/
    systemctl daemon-reload
    echo "✓ Systemd service installed"
    echo "  Run: systemctl enable categraf"
    echo "  Run: systemctl start categraf"
fi

# 添加到 PATH
if ! grep -q "/usr/local/categraf/bin" /etc/profile 2>/dev/null; then
    echo 'export PATH=$PATH:/usr/local/categraf/bin' >> /etc/profile
    echo "✓ Added to PATH (reload shell or run: source /etc/profile)"
fi

echo "✓ Categraf installed to /usr/local/categraf"
echo ""
echo "Configuration files: /usr/local/categraf/conf/"
echo "Edit /usr/local/categraf/conf/config.toml before starting"

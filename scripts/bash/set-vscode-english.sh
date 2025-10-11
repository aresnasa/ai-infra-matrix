#!/bin/bash

# VS Code 语言设置脚本
echo "🔧 正在设置 VS Code 为英语..."

# VS Code 可执行文件路径
VSCODE_PATH="/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"

# 检查 VS Code 是否安装
if [ ! -f "$VSCODE_PATH" ]; then
    echo "❌ 未找到 VS Code，请确保 VS Code 已安装在 /Applications/ 目录下"
    exit 1
fi

echo "✅ 找到 VS Code"

# 禁用中文语言包
echo "📦 禁用中文语言包..."
"$VSCODE_PATH" --disable-extension ms-ceintl.vscode-language-pack-zh-hans

# 设置用户级别的语言配置
USER_SETTINGS_DIR="$HOME/Library/Application Support/Code/User"
USER_SETTINGS_FILE="$USER_SETTINGS_DIR/settings.json"

echo "📝 更新用户设置..."

# 创建用户设置目录（如果不存在）
mkdir -p "$USER_SETTINGS_DIR"

# 检查设置文件是否存在
if [ ! -f "$USER_SETTINGS_FILE" ]; then
    # 创建新的设置文件
    echo '{
    "locale": "en"
}' > "$USER_SETTINGS_FILE"
    echo "✅ 创建了新的用户设置文件"
else
    echo "ℹ️  用户设置文件已存在，请手动添加 \"locale\": \"en\" 到设置中"
fi

echo "🎉 设置完成！"
echo ""
echo "📋 下一步操作："
echo "1. 重启 VS Code"
echo "2. 如果界面仍然是中文，请："
echo "   - 按 Cmd+Shift+P 打开命令面板"
echo "   - 输入 'Configure Display Language'"
echo "   - 选择 'English (en)'"
echo "   - 重启 VS Code"
echo ""
echo "🗑️  如需完全移除中文语言包，运行："
echo "   $VSCODE_PATH --uninstall-extension ms-ceintl.vscode-language-pack-zh-hans"
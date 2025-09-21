
## 当前状态
✅ 工作区语言已设置为英语：`"locale": "en"`

## 设置 VS Code 显示语言为英语

### 方法一：通过命令面板（推荐）
1. 打开 VS Code
2. 按 `Cmd+Shift+P` (macOS) 打开命令面板
3. 输入 `Configure Display Language`
4. 选择 `English (en)`
5. 重启 VS Code

### 方法二：通过设置界面
1. 打开 VS Code
2. 按 `Cmd+,` 打开设置
3. 搜索 `locale`
4. 将 `Locale` 设置为 `en`
5. 重启 VS Code

### 方法三：编辑用户设置文件
如果需要在全局用户设置中确保英语语言：

1. 按 `Cmd+Shift+P` 打开命令面板
2. 输入 `Preferences: Open Settings (JSON)`
3. 添加或确保有以下设置：

```json
{
    "locale": "en"
}
```

## 相关设置项

### locale
- `"en"`：英语
- `"zh-cn"`：简体中文
- `"ja"`：日语
- `"fr"`：法语
- 等等...

### 验证设置
重启 VS Code 后，界面应该显示为英语。如果仍然显示中文，请检查：

1. 是否已安装中文语言包扩展
2. 系统语言设置
3. VS Code 版本是否支持语言切换

## 移除中文语言包（可选）
如果你想完全移除中文界面：

1. 打开扩展面板 (`Cmd+Shift+X`)
2. 搜索已安装的中文语言包（通常是 "Chinese (Simplified)"）
3. 点击禁用或卸载
4. 重启 VS Code

## 故障排查
如果设置后语言没有改变：

1. 确认已重启 VS Code
2. 检查是否有语言包扩展覆盖了设置
3. 检查系统语言环境变量
4. 尝试重新安装 VS Code
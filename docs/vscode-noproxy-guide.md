# VS Code NoProxy 配置说明

## 已配置内容

1. **工作区设置**：已更新 `.vscode/settings.json` 文件，添加了 noproxy 配置
2. **配置模板**：创建了 `vscode-noproxy-clean.json` 文件，包含完整的 noproxy 配置

## 配置说明

### 工作区级别配置
- 文件位置：`.vscode/settings.json`
- 作用范围：仅当前工作区
- 已自动配置完成

### 全局用户配置（需要手动操作）
要在全局范围内应用 noproxy 设置，请按以下步骤操作：

1. 打开 VS Code
2. 使用快捷键 `Cmd+Shift+P` 打开命令面板
3. 输入 "Preferences: Open Settings (JSON)"
4. 将 `vscode-noproxy-clean.json` 文件中的内容合并到你的用户设置中

或者直接编辑文件：
```bash
open ~/Library/Application\ Support/Code/User/settings.json
```

## 配置项说明

### http.proxySupport
- `on`：启用代理支持
- `off`：禁用代理支持
- `fallback`：如果直接连接失败则使用代理

### http.noProxy
配置不使用代理的地址列表，包括：

- **本地地址**：localhost, 127.0.0.1, ::1
- **私有网络**：10.*, 192.168.*, 172.16.* - 172.31.*
- **本地域名**：*.local, *.internal, *.intranet
- **Kubernetes**：*.cluster.local, *.svc.cluster.local
- **开发环境**：*.dev, *.test, *.staging

## 自定义配置

如果你需要添加特定的域名或IP地址到 noproxy 列表：

1. 编辑 `.vscode/settings.json`
2. 在 `http.noProxy` 数组中添加你的地址
3. 保存文件

示例：
```json
{
    "http.noProxy": [
        "localhost",
        "127.0.0.1",
        "your-custom-domain.com",
        "192.168.100.50"
    ]
}
```

## 验证配置

重启 VS Code 后，配置会自动生效。你可以通过以下方式验证：

1. 查看 VS Code 的输出面板
2. 检查网络请求是否正常
3. 观察扩展下载和更新是否正常工作

## 故障排查

如果遇到问题：

1. 检查 JSON 语法是否正确
2. 确认代理设置是否与系统代理冲突
3. 重启 VS Code
4. 检查系统环境变量（NO_PROXY, HTTP_PROXY 等）
# AI 配置测试使用指南

## 🎯 快速开始

### 1. 配置环境变量

有三种方式配置 API Key：

#### 方式 1：临时环境变量（推荐用于测试）

```bash
export DEEPSEEK_API_KEY="sk-your-real-api-key-here"
./run-ai-config-test.sh
```

#### 方式 2：创建 .env.local 文件（推荐用于开发）

```bash
# 复制模板文件
cp .env.test.example .env.local

# 编辑 .env.local，填入真实的 API Key
# DEEPSEEK_API_KEY=sk-your-real-api-key-here
# BASE_URL=http://192.168.0.200:8080
# HEADED=false

# 加载环境变量并运行测试
source .env.local
./run-ai-config-test.sh
```

#### 方式 3：永久环境变量（推荐用于个人开发机）

```bash
# 添加到 ~/.zshrc 或 ~/.bashrc
echo 'export DEEPSEEK_API_KEY="sk-your-real-api-key-here"' >> ~/.zshrc

# 重新加载配置
source ~/.zshrc

# 运行测试
./run-ai-config-test.sh
```

### 2. 运行测试

```bash
# 运行完整的 E2E 测试
./run-ai-config-test.sh

# 仅运行 API 测试
node test-ai-config-api.js
```

## 🔒 安全最佳实践

### ✅ 正确做法

1. **使用环境变量**
   ```javascript
   const apiKey = process.env.DEEPSEEK_API_KEY;
   ```

2. **配置 .gitignore**
   ```
   .env.local
   .env.test
   ```

3. **使用配置模板**
   - 提交：`.env.test.example`（不含真实密钥）
   - 不提交：`.env.local`（包含真实密钥）

4. **日志脱敏**
   ```javascript
   console.log(`API Key: ${apiKey.substring(0, 10)}...`);
   ```

### ❌ 错误做法

1. **硬编码 API Key**
   ```javascript
   // ❌ 绝对不要这样做！
   const apiKey = 'sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx';
   ```

2. **提交敏感文件**
   ```bash
   # ❌ 不要提交这些文件
   git add .env.local
   git add .env.test
   ```

3. **明文日志输出**
   ```javascript
   // ❌ 不要输出完整的 API Key
   console.log(`API Key: ${apiKey}`);
   ```

## 🛡️ 安全检查清单

在提交代码前，请运行安全检查：

```bash
./check-security.sh
```

检查项目包括：
- ✅ 硬编码的 API Key
- ✅ 硬编码的密码
- ✅ .gitignore 配置
- ✅ 敏感文件是否被 Git 跟踪
- ✅ 环境变量使用情况

## 📁 相关文件

- `test/e2e/specs/ai-config-test.spec.js` - Playwright E2E 测试
- `test-ai-config-api.js` - API 直接测试
- `run-ai-config-test.sh` - 测试运行脚本
- `.env.test.example` - 环境变量配置模板
- `check-security.sh` - 安全检查脚本

## 🐛 故障排查

### 问题 1：提示 API Key 未设置

**错误信息：**
```
❌ 错误: DEEPSEEK_API_KEY 环境变量未设置
```

**解决方案：**
```bash
# 检查环境变量是否设置
echo $DEEPSEEK_API_KEY

# 如果为空，请设置环境变量
export DEEPSEEK_API_KEY="sk-your-real-key"
```

### 问题 2：测试超时

**可能原因：**
- 网络连接问题
- API 服务不可用
- 超时设置过短

**解决方案：**
```bash
# 检查网络连接
curl http://192.168.0.200:8080/health

# 增加超时时间（修改测试文件中的 timeout 设置）
```

### 问题 3：API Key 无效

**错误信息：**
```
401 Unauthorized
```

**解决方案：**
1. 检查 API Key 是否正确
2. 检查 API Key 是否已过期
3. 访问 https://platform.deepseek.com/api_keys 重新生成

## 📝 环境变量说明

| 变量名 | 必需 | 默认值 | 说明 |
|--------|------|--------|------|
| `DEEPSEEK_API_KEY` | ✅ | - | DeepSeek API 密钥 |
| `BASE_URL` | ❌ | `http://192.168.0.200:8080` | 测试服务器地址 |
| `HEADED` | ❌ | `false` | 是否显示浏览器窗口 |

## 🔄 Git 历史清理

如果不小心将 API Key 提交到了 Git 历史中，请执行以下步骤：

### 1. 立即失效泄露的 API Key

访问 https://platform.deepseek.com/api_keys 删除或重新生成 API Key

### 2. 清理 Git 历史

```bash
# 方法 1：重置到安全的提交点（如果是最近的提交）
git reset --hard <safe-commit-hash>

# 方法 2：使用 git filter-branch（适用于历史提交）
git filter-branch --tree-filter 'git ls-files -z | xargs -0 sed -i "s/sk-[a-zA-Z0-9]\{32,\}/sk-REDACTED/g"' HEAD

# 方法 3：使用 git-filter-repo（推荐）
pip install git-filter-repo
git filter-repo --path-match '*.js' --replace-text <(echo 'sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx==>sk-REDACTED')
```

### 3. 强制推送（谨慎操作）

```bash
# ⚠️ 警告：这会覆盖远程历史
git push --force-with-lease origin v0.3.8
```

### 4. 通知团队成员

```bash
# 所有团队成员需要重新克隆或同步
git fetch origin
git reset --hard origin/v0.3.8
```

## 📚 相关文档

- [Playwright 测试文档](https://playwright.dev/)
- [DeepSeek API 文档](https://platform.deepseek.com/docs)
- [Git 安全最佳实践](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure)

# ocgt - Claude API 兼容性代理控制面板

`ocgt` 是一个轻量级的 Claude API 兼容性代理工具，内置 **Web可视化控制面板**，用于连接 Claude Code 和 OpenCode Go API。

## 核心特性

- **Web可视化控制面板** - 内置现代化暗色主题UI，无需额外安装
- **实时流量监控** - 可视化查看所有代理请求的历史记录
- **一键配置管理** - 在网页中切换Profile、更新API Key
- **环境配置助手** - 自动生成并复制Claude Code环境变量命令
- **CC Switch集成** - 一键生成CC Switch提供商配置JSON
- **多模型支持** - 支持Kimi、GLM、DeepSeek、Qwen、MiMo、MiniMax等国产大模型
- **协议转换** - 自动将Anthropic Messages协议转换为OpenAI Chat Completions协议

## 架构概览

```
Claude Code -> ocgt 本地代理 -> OpenCode Go 官方 API
                    ↓
          Web控制面板 (http://127.0.0.1:8787)
```

## 快速开始

### 1. 编译

```powershell
git clone https://github.com/ethan-blue/open-code-go-tools.git
cd open-code-go-tools
go build -o .\bin\ocgt.exe .\cmd\ocgt
```

### 2. 初始化配置

```powershell
.\bin\ocgt.exe init
```

默认配置文件路径：`%USERPROFILE%\.ocgt\config.json`

### 3. 设置 API 密钥

```powershell
# 方式一：通过命令行
.\bin\ocgt.exe key set "your-opencode-go-key"

# 方式二：通过Web控制面板（推荐）
# 启动代理后访问 http://127.0.0.1:8787 直接在网页中输入密钥
```

### 4. 启动代理服务

```powershell
.\bin\ocgt.exe serve
```

### 5. 打开 Web 控制面板

浏览器访问：`http://127.0.0.1:8787`

在Web面板中你可以：
- 查看系统运行状态
- 切换活跃Profile
- 更新API Key
- 查看实时请求历史
- 复制Claude Code环境配置命令
- 生成CC Switch配置JSON

### 6. 配置 Claude Code

在Web面板的"环境配置助手"卡片中，选择你的终端类型（PowerShell/Bash/CMD），点击"复制命令"，然后在开发终端中执行。

或者在另一个终端中运行：

```powershell
.\bin\ocgt.exe claude-env
```

执行输出的命令后启动 Claude Code：

```powershell
claude
```

## Web 控制面板功能

### 系统状态总览
- 监听地址
- 上游API地址
- 活跃Profile
- 默认大模型
- 请求超时设置
- 配置文件路径

### 配置与密钥管理
- Profile下拉切换
- API Key输入与保存（直接写入config.json）
- 密钥明密文切换显示

### Claude Code 环境配置
- 支持PowerShell/Bash/CMD三种终端
- 一键复制环境变量命令
- 自动根据当前配置生成

### CC Switch 提供商配置
- 一键生成Provider JSON配置
- 自动填充当前Profile信息
- 一键复制到剪贴板

### 实时流量监控
- 实时显示代理请求历史
- 包含：时间、方法、路径、目标模型、HTTP状态、响应时间
- 每2.5秒自动刷新

## API 端点

### 代理端点
| 端点 | 方法 | 说明 |
|------|------|------|
| `/v1/models` | GET | 获取模型列表 |
| `/v1/messages` | POST | 发送消息（Anthropic协议） |
| `/v1/messages/count_tokens` | POST | 计算token数 |
| `/healthz` | GET | 健康检查 |
| `/ocgt/profile` | GET | 获取当前Profile |

### Web面板API
| 端点 | 方法 | 说明 |
|------|------|------|
| `/ocgt/api/status` | GET | 获取系统状态 |
| `/ocgt/api/profiles` | GET | 获取所有Profile列表 |
| `/ocgt/api/profiles/active` | POST | 切换活跃Profile |
| `/ocgt/api/key` | POST | 更新API Key |
| `/ocgt/api/history` | GET | 获取请求历史 |

## 命令行工具

| 命令 | 说明 |
|------|------|
| `ocgt init` | 创建配置文件 |
| `ocgt serve` | 启动本地代理服务 |
| `ocgt profiles` | 列出已配置的profiles |
| `ocgt models` | 显示本地模型别名或使用 `--remote` 查询官方模型列表 |
| `ocgt claude-env` | 打印 Claude Code 环境变量配置 |
| `ocgt ccswitch` | 打印 CC Switch 提供商配置片段 |
| `ocgt key set <key>` | 保存 API Key 到用户环境变量 |
| `ocgt key show` | 显示当前设置的 API Key |
| `ocgt version` | 显示版本号 |

## 配置说明

### 默认配置

```json
{
  "listen": "127.0.0.1:8787",
  "upstream": "https://opencode.ai/zen/go",
  "request_timeout_seconds": 300,
  "active_profile": "opencode-go",
  "profiles": {
    "opencode-go": {
      "api_key_env": "OPENCODE_GO_API_KEY",
      "default_model": "kimi-k2.6",
      "model_aliases": {
        "deepseek": "deepseek-v4-pro",
        "flash": "deepseek-v4-flash",
        "glm": "glm-5.1",
        "glm5": "glm-5",
        "haiku": "deepseek-v4-flash",
        "hy3": "hy3-preview",
        "kimi": "kimi-k2.6",
        "kimi25": "kimi-k2.5",
        "mimo": "mimo-v2.5-pro",
        "mimo25": "mimo-v2.5",
        "minimax": "minimax-m2.7",
        "opus": "kimi-k2.6",
        "qwen35": "qwen3.5-plus",
        "qwen": "qwen3.6-plus",
        "sonnet": "qwen3.6-plus"
      },
      "message_models": [
        "minimax-m2.5",
        "minimax-m2.7"
      ]
    }
  }
}
```

### 配置项说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `listen` | 本地代理监听地址 | `127.0.0.1:8787` |
| `upstream` | OpenCode Go API 上游地址 | `https://opencode.ai/zen/go` |
| `request_timeout_seconds` | 请求超时时间（秒） | `300` |
| `active_profile` | 默认使用的配置 profile | `opencode-go` |
| `api_key_env` | API Key 环境变量名称 | `OPENCODE_GO_API_KEY` |
| `default_model` | 默认模型 | `kimi-k2.6` |
| `model_aliases` | 模型别名映射 | 见上表 |
| `message_models` | 使用 Messages 端点的模型 | `minimax-m2.5`, `minimax-m2.7` |

## 支持的模型

### OpenCode Go 官方模型

- `glm-5`, `glm-5.1`
- `kimi-k2.5`, `kimi-k2.6`
- `mimo-v2.5`, `mimo-v2.5-pro`
- `minimax-m2.5`, `minimax-m2.7`
- `qwen3.5-plus`, `qwen3.6-plus`
- `deepseek-v4-pro`, `deepseek-v4-flash`
- `hy3-preview`

### 模型别名

| 别名 | 映射模型 | 说明 |
|------|----------|------|
| `kimi` | `kimi-k2.6` | 默认模型 |
| `kimi25` | `kimi-k2.5` | Kimi上一代 |
| `glm` | `glm-5.1` | 智谱最新 |
| `glm5` | `glm-5` | 智谱5代 |
| `qwen` | `qwen3.6-plus` | 通义最新 |
| `qwen35` | `qwen3.5-plus` | 通义3.5 |
| `deepseek` | `deepseek-v4-pro` | 深度求索Pro |
| `flash` | `deepseek-v4-flash` | 深度求索Flash |
| `mimo` | `mimo-v2.5-pro` | 面壁Pro |
| `mimo25` | `mimo-v2.5` | 面壁标准 |
| `minimax` | `minimax-m2.7` | MiniMax最新 |
| `opus` | `kimi-k2.6` | 对标Claude Opus |
| `sonnet` | `qwen3.6-plus` | 对标Claude Sonnet |
| `haiku` | `deepseek-v4-flash` | 对标Claude Haiku |
| `hy3` | `hy3-preview` | Hy3预览版 |

## CC Switch 配置

在Web面板中点击"CC Switch 提供商配置"卡片，复制生成的JSON，然后导入CC Switch客户端。

或使用命令行生成：

```powershell
.\bin\ocgt.exe ccswitch --profile opencode-go
```

### 推荐的 CC Switch 映射

| 角色 | 菜单名称 | 实际模型 |
|------|----------|----------|
| Opus | Kimi K2.6 | `kimi-k2.6` |
| Sonnet | GLM-5.1 | `glm-5.1` |
| Sonnet | Qwen3.6 Plus | `qwen3.6-plus` |
| Sonnet | DeepSeek V4 Pro | `deepseek-v4-pro` |
| Haiku | DeepSeek V4 Flash | `deepseek-v4-flash` |
| Sonnet | MiniMax M2.7 | `minimax-m2.7` |
| Haiku | MiniMax M2.5 | `minimax-m2.5` |

## 技术细节

### 协议转换流程

1. **Claude Code 请求** -> Anthropic Messages 格式
2. **ocgt 代理** -> 根据模型类型决定路由：
   - Messages 模型 (MiniMax) -> 直接转发到 `/v1/messages`
   - Chat Completions 模型 (Kimi/GLM/DeepSeek/Qwen) -> 转换为 OpenAI 格式并转发到 `/v1/chat/completions`
3. **OpenCode Go API** -> 返回响应
4. **ocgt 代理** -> 将响应转换回 Anthropic Messages 格式
5. **Claude Code** <- 接收标准 Anthropic 响应

### 支持的特性

- 流式响应 (Streaming)
- 工具调用 (Tool Calling)
- DeepSeek 思考模式兼容
- 多模型支持
- 模型别名映射
- 自定义 Headers
- 多Profile支持
- Web可视化管理

## 项目结构

```
open-code-go-tools/
├── cmd/ocgt/main.go           # CLI 主程序入口
├── internal/
│   ├── config/
│   │   ├── config.go          # 配置管理
│   │   └── config_test.go     # 配置测试
│   └── proxy/
│       ├── types.go           # 类型定义和 Server 构造
│       ├── converter.go       # Anthropic/OpenAI 协议转换
│       ├── streamer.go        # SSE 流式响应处理
│       ├── handler.go         # HTTP 路由和处理逻辑
│       ├── helpers.go         # 工具函数
│       ├── web_handler.go     # Web静态文件服务
│       ├── proxy_test.go      # 代理测试
│       └── web/               # Web控制面板前端
│           ├── index.html     # 主页面
│           ├── app.js         # 前端逻辑
│           └── style.css      # 样式表
├── app.go                     # GUI 应用集成
├── env_windows.go             # Windows 环境变量设置
├── env_other.go               # 非Windows平台存根
├── go.mod                     # Go 模块定义
├── Makefile                   # 构建脚本
├── LICENSE                    # MIT 许可证
└── README.md                  # 项目说明
```

## 参考文档

- [OpenCode Go 文档](https://dev.opencode.ai/docs/go/)
- [Claude Code 环境变量](https://code.claude.com/docs/en/env-vars)
- [CC Switch](https://cc-switch.cc/en)

## 许可证

MIT License
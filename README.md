# ocgt - Claude Code 本地代理与 GUI 控制面板

`ocgt` (OpenCode Go Tools) 是面向 Claude Code 的本地 API 兼容代理。它把 Claude Code 的 Anthropic Messages 请求转换为 OpenCode Go 上游可用的接口，并提供一个 Wails 原生 GUI：保存密钥、切换模型映射、修复环境变量、拉起已配置终端、查看实时请求日志，都可以在图形界面里完成。

> 当前 Release 工作流构建的是 GUI 客户端：Windows 产物为 `ocgt-windows-amd64.exe`，macOS/Linux 产物为对应平台的 `ocgt-*` 可执行文件。下载后直接运行即可使用图形界面。

## GUI 预览

### 系统状态

![ocgt 系统状态监控](assets/gui_status.png)

启动后会自动加载本地代理状态，显示监听地址、上游 API、当前 Profile、默认模型、API Key 是否已配置，以及本地 `config.json` 路径。

### 配置管理

![ocgt 配置管理](assets/gui_config.png)

在 GUI 中选择 Profile，填写 OpenCode Go API Key，选择默认模型与 Sonnet/Haiku/Opus 映射目标，然后点击“保存并热重载配置”。保存成功后服务会即时应用新配置。

### 一键终端

![ocgt 终端启动](assets/gui_terminal.png)

选择 PowerShell、CMD 或 Bash 后点击“一键拉起配置终端”，新终端会带上当前代理环境变量。进入终端后直接运行 `claude` 即可。

## 为什么会看到旧模型

如果 `/model` 里已经能看到 `kimi-k2.6`，但同时还出现旧的 `astron-code-latest`、旧 Sonnet/Opus/Haiku 名称，通常不是 ocgt 解析错了，而是 Claude Code 或 CC Switch 之前写入过这些环境变量：

- `ANTHROPIC_DEFAULT_SONNET_MODEL_NAME`
- `ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME`
- `ANTHROPIC_DEFAULT_OPUS_MODEL_NAME`
- `ANTHROPIC_DEFAULT_SONNET_MODEL`
- `ANTHROPIC_DEFAULT_HAIKU_MODEL`
- `ANTHROPIC_DEFAULT_OPUS_MODEL`
- `ANTHROPIC_AUTH_TOKEN`

GUI 的“一键修复 Claude Code 系统环境变量”会清理这些旧变量，并只保留 ocgt 当前需要的代理变量。修复后请关闭旧 PowerShell/CMD，再打开新终端运行 `claude`，否则旧进程仍可能继承老环境。

## 502 是否需要考虑

需要。流量监控里的 502/504 表示 ocgt 已经收到 Claude Code 请求，但连接上游或等待上游响应失败。常见原因包括：

- 上游服务短时不可用或限流。
- API Key 错误、额度不足或上游拒绝。
- 本机网络、代理或 DNS 问题。
- 请求耗时超过 `request_timeout_seconds`。

GUI 的“流量监控”会显示成功率、平均延时，并在失败行中展示上游返回的错误原因。502 不会被伪装成成功，这样才能看清真实稳定性。

## 快速开始

1. 到 [Releases](../../releases) 下载最新 GUI 可执行文件。
2. 双击运行 `ocgt-windows-amd64.exe`，程序会启动本地代理，默认监听 `127.0.0.1:8787`。
3. 在“配置管理”里填写 OpenCode Go API Key，保存并热重载。
4. 点击“一键修复 Claude Code 系统环境变量”，清理旧 CC Switch/Claude Code 残留。
5. 到“终端启动”页点击“一键拉起配置终端”，在新终端里运行 `claude`。

## 开发构建

需要 Go 1.22+ 和 Wails CLI。

```powershell
# 构建 GUI 版本，输出到 build\bin\ocgt.exe
wails build

# 构建极简 CLI 版本，输出到 bin\ocgt.exe
go build -o .\bin\ocgt.exe .\cmd\ocgt
```

## 配置文件

默认配置路径：

```text
%USERPROFILE%\.ocgt\config.json
```

示例：

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
        "kimi": "kimi-k2.6",
        "opus": "kimi-k2.6",
        "sonnet": "qwen3.6-plus",
        "haiku": "deepseek-v4-flash",
        "qwen": "qwen3.6-plus",
        "deepseek": "deepseek-v4-pro",
        "glm": "glm-5.1"
      }
    }
  }
}
```

## 模型映射

| Claude Code 请求 | 默认上游模型 | 用途 |
| :--- | :--- | :--- |
| `opus` / 含 `opus` 的 Claude 模型名 | `kimi-k2.6` | 复杂推理、重构、长上下文 |
| `sonnet` / 含 `sonnet` 的 Claude 模型名 | `qwen3.6-plus` | 日常编码、较均衡的速度与质量 |
| `haiku` / 含 `haiku` 的 Claude 模型名 | `deepseek-v4-flash` | 快速、低成本请求 |
| `kimi` | `kimi-k2.6` | 直接指定 Kimi |

## CLI

```powershell
ocgt init
ocgt serve
ocgt models
ocgt claude-env
ocgt ccswitch
```

`ocgt ccswitch` 会输出可导入 CC Switch 的 provider JSON。推荐只保留一个 `ocgt-*` provider，旧的 astron provider 可以在 CC Switch 里删除，避免 `/model` 菜单继续显示历史选项。

## License

MIT

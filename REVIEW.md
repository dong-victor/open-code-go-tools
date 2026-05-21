# ocgt 项目评审报告

## 总体评价

ocgt 是一个功能明确、结构清晰的 Go 代理工具，核心职责是把 Anthropic Messages 协议和 OpenAI Chat Completions 协议互转，让 Claude Code 通过 OpenCode Go 的 API 使用国产大模型。

**最新版本特性：**
- 内置 Web 可视化控制面板（暗色主题、霓虹风格）
- 实时流量监控（每2.5秒自动刷新）
- 可视化 Profile 切换和 API Key 管理
- 一键生成 Claude Code 环境配置命令
- 一键生成 CC Switch 配置 JSON
- GUI 应用集成（Wails）
- 代码量约 2500+ 行

以下按类别列出问题和改进建议。

---

## 一、安全性问题

### P0 - 关键问题

**1. 本地代理无认证机制** (未修复 - 低优先级)

`serve` 命令监听 `127.0.0.1:8787`，任何本地进程都能调用。虽然 API Key 存在上游验证，但本地无鉴权意味着同机恶意软件可直接使用代理并发请求到 OpenCode Go，消耗订阅额度。

建议：添加可选的本地 Bearer Token 认证，通过配置 `local_auth_token` 或环境变量 `OCGT_LOCAL_TOKEN` 控制。

**2. 请求体大小限制** ✅ 已修复

`io.ReadAll(r.Body)` 已替换为 `io.ReadAll(io.LimitReader(r.Body, MaxBodySize))`，限制为 10MB。常量 `MaxBodySize` 定义在 `types.go` 中。

**3. API Key 掩码显示** ✅ 已修复

`maskKey` 已改为只显示末尾 4 字符 + 总长度，如 `****abcd (44 chars)`。

```go
// main.go
func maskKey(key string) string {
    if len(key) <= 4 {
        return strings.Repeat("*", len(key))
    }
    return "****" + key[len(key)-4:] + fmt.Sprintf(" (%d chars)", len(key))
}
```

**4. 上游 API Key 传输安全**

`key set` 使用 PowerShell 命令行参数设置环境变量，在进程列表中可见明文 Key。

建议：通过 `stdin` 或临时文件传递值，而非命令行参数。Web面板中直接输入密钥是更安全的替代方案。

### P1 - 中等问题

**5. 上游 TLS 证书未校验**

`http.Transport` 使用默认配置，没有自定义 TLS 校验逻辑。

建议：添加配置项 `insecure_skip_verify` 以便在自签名环境使用。

**6. 无 HTTP 请求速率限制**

代理不限制并发请求数或请求频率。

建议：添加可选的并发请求限制。

---

## 二、错误处理

### P1 - 中等问题

**1. 流式 SSE 写入错误处理** ✅ 已改进

`writeSSE` 函数现在返回 error，`sendSSE` 包装器会记录错误日志：

```go
// streamer.go
func sendSSE(w io.Writer, event string, payload any) {
    if err := writeSSE(w, event, payload); err != nil {
        log.Printf("SSE write error: event=%s: %v", event, err)
    }
}
```

**2. 上游错误响应体可能泄露内部信息**

`writeUpstreamError` 直接将上游的错误响应体原样返回客户端。

建议：解析上游错误并封装为标准格式后再返回。

**3. `copyResponse` 中写入错误处理** ✅ 已改进

现在在写入错误时提前终止：

```go
// helpers.go
if writeErr != nil {
    return written, writeErr
}
```

**4. `models` 端点错误时无降级**

如果上游 `/v1/models` 返回非 200，直接透传错误给客户端。已实现本地 fallback 模型列表（`configuredModels` 函数）。

---

## 三、代码质量

### P2 - 低优先级

**1. Token 估算优化** ✅ 已改进

`estimateTokens` 现在区分 CJK 和 ASCII 字符：

```go
// helpers.go
for _, r := range text {
    if r > 127 {
        tokenEstimate += 3 // CJK characters roughly 2-3 tokens each
    } else {
        tokenEstimate++ // ASCII characters, ~4 per token
    }
}
```

**2. `fetchRemoteModels` 使用自定义 Client** ✅ 已修复

现在使用带 30 秒超时的自定义 Client，而非 `http.DefaultClient`。

**3. 日志缺乏结构化**

使用 `log.Printf` 裸字符串，无请求 ID、无时间戳格式控制、无日志级别。

建议：至少添加请求 ID 或使用 `slog`（Go 1.21+ 标准库）。

**4. Windows 特定代码使用构建标签** ✅ 已修复

`env_windows.go` 和 `env_other.go` 使用 `//go:build windows` 和 `//go:build !windows` 分离。

**5. 版本号改为 var** ✅ 已修复

`main.go` 中 `version` 已从 `const` 改为 `var`，支持构建时注入：

```go
var version = "0.1.1"
```

---

## 四、测试覆盖

### P0 - 关键缺失

**1. 流式响应测试**

`streamer.go` 是最复杂的部分（293 行），但没有任何单元测试。

建议：添加测试用例覆盖：
- 纯文本流式场景
- reasoning_content（思考）流式场景
- 工具调用流式场景
- 混合文本+工具调用流式场景
- `[DONE]` 终止信号处理

**2. 无并发安全测试**

`reasoningByTool` 的 `sync.Mutex` 在并发场景下的正确性未测试。

**3. 缺少集成测试**

现有测试使用 `httptest.NewServer` 模拟上游，但缺少端到端集成测试。

---

## 五、性能问题

### P1 - 中等问题

**1. `reasoningOrder` 切片的切片操作**

每次 LRU 驱逐都会切片再切片，高频场景下产生大量 GC 压力。

建议：使用环形缓冲区或 `container/list` + map 实现 LRU。

**2. 无 gzip 请求压缩**

代理不设置 `Accept-Encoding: gzip` 请求头。

建议：添加 gzip 支持。

**3. Transport 配置**

Transport 设置了 `IdleConnTimeout: 90s`，但没有 `MaxConnsPerHost` 限制。

---

## 六、协议兼容性

### P1 - 中等问题

**1. Anthropic `system` 字段处理**

`blocksToText` 将 Anthropic 的数组格式 system 消息扁平化为单个字符串，会丢失 `cache_control` 标记。

**2. 图片处理使用 Markdown 格式**

将 Anthropic 的 base64 图片转为 Markdown 图片语法。大多数 Chat Completions 模型不支持这种格式。

建议：使用 OpenAI 的 `image_url` 多模态格式。

**3. `count_tokens` 端点**

虽然已改进 CJK 估算，但仍然是估算值，非精确计算。

---

## 七、架构与设计

### P1 - 中等问题

**1. 多 Profile 支持** ✅ 已完善

Config 支持 `profiles` 和 `active_profile`，Web 面板支持动态切换 Profile 并保存到配置文件。

**2. 无配置文件热重载**

修改 config.json 后需重启代理才能生效。Web 面板中修改 API Key 会直接写入配置文件，但其他配置仍需重启。

建议：添加 SIGHUP 信号处理或 `/ocgt/reload` 端点。

**3. CORS 支持** ✅ 已实现

`requestLogger` 中间件已添加 CORS 支持，允许前端 API 请求：

```go
// helpers.go
w.Header().Set("Access-Control-Allow-Origin", origin)
w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, ...")
```

**4. 优雅关闭日志** ✅ 已改进

`ListenAndServe` 现在有日志提示正在关闭：

```go
log.Println("shutting down...")
// ...
log.Println("server stopped")
```

---

## 八、新增功能（Web 控制面板）

### 亮点

1. **现代化 UI** - 暗色主题、霓虹风格、Glassmorphism 卡片设计
2. **实时状态监控** - 系统状态、Profile 管理、API Key 管理
3. **流量历史表格** - 实时显示请求记录，包含时间、方法、路径、模型、状态、响应时间
4. **环境配置助手** - 支持 PowerShell/Bash/CMD 三种终端，一键复制
5. **CC Switch 集成** - 一键生成并复制 Provider JSON 配置
6. **响应式设计** - 适配移动端和桌面端

### 建议改进

1. **添加认证** - Web 面板应添加可选的访问密码
2. **添加请求统计** - 显示今日/本周/本月请求总数和 token 消耗
3. **添加模型使用统计** - 显示各模型的使用频率
4. **添加错误日志面板** - 显示代理错误和上游错误
5. **添加配置导出/导入** - 支持备份和恢复配置

---

## 九、文档问题

1. **README 已更新** ✅ - 现在包含 Web 控制面板的详细说明
2. **缺少 CHANGELOG** - 无版本变更记录
3. **缺少 CONTRIBUTING.md** - 无贡献指南
4. **Web 面板缺少单独文档** - 建议添加 Web 面板功能说明和使用指南

---

## 十、改进优先级建议

| 优先级 | 改进项 | 工作量 |
|--------|--------|--------|
| P0 | 流式响应单元测试 | 中 |
| P0 | Web 面板访问认证（可选） | 小 |
| P1 | 上游错误响应封装 | 小 |
| P1 | 添加 gzip 压缩支持 | 中 |
| P1 | 结构化日志（slog） | 中 |
| P2 | 配置热重载 | 中 |
| P2 | 请求统计面板 | 中 |
| P2 | 错误日志面板 | 小 |

---

## 总结

ocgt 作为一个本地代理工具，核心功能（协议转换、流式桥接、DeepSeek 思考模式兼容）实现完整且经过实际使用验证。**最新版本最大的亮点是内置了精美的 Web 可视化控制面板**，让用户可以通过浏览器轻松管理代理配置、查看实时流量、一键生成环境配置。

主要优势：
1. **Web 可视化** - 现代化暗色主题UI，功能完整
2. **实时流量监控** - 每2.5秒自动刷新请求历史
3. **配置管理便捷** - Profile切换、API Key更新都在网页中完成
4. **环境配置助手** - 支持多种终端，一键复制命令
5. **代码结构清晰** - 模块划分合理，使用构建标签分离平台特定代码

主要短板：
1. **安全性** - 缺少本地认证和 Web 面板访问控制
2. **测试** - 最核心的流式转换逻辑完全没有测试
3. **协议完整性** - system 处理、图片格式等有已知限制
4. **文档** - 缺少 CHANGELOG 和贡献指南

建议优先处理 P0 问题（测试、Web认证），然后逐步改进 P1（错误处理、gzip）。
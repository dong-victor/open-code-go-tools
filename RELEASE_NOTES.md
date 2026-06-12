# Release Notes

## 🌐 语言选择 / Language
* [简体中文 (Simplified Chinese)](#-ocgt-v204---v204-)
* [English](#-ocgt-v204---release-notes)

---

# 🇨🇳 ocgt v2.0.4

## 修复

- **`/v1/models` 端点缺少认证头**：查询模型列表时未附带 `X-Api-Key` / `Anthropic-Version` 头，导致部分上游网关返回 401。修复后与 `/v1/messages` 端点认证行为一致
- **`/v1/chat/completions` 转发缺少 `X-Api-Key`**：通过 chat/completions 路径转发请求时仅携带 `Authorization: Bearer`，未附加 `X-Api-Key`。部分上游（如 opencode.ai 网关）要求 `X-Api-Key` 头，缺失会导致 401 认证失败。修复后同时发送 `X-Api-Key` 和 `Anthropic-Version` 头
- **新增测试覆盖**：补充 `TestChatCompletionsEndpointUsesAnthropicAuth` 测试，确保 chat/completions 路径的认证头行为正确

---

# 🇺🇸 ocgt v2.0.4 - Release Notes

## Fixes

- **Missing auth headers on `/v1/models`**: Model list queries were not sending `X-Api-Key` / `Anthropic-Version` headers, causing 401 errors on some upstream gateways. Fixed to match `/v1/messages` auth behavior
- **Missing `X-Api-Key` on `/v1/chat/completions`**: Requests forwarded via chat/completions path only carried `Authorization: Bearer` without `X-Api-Key`. Some upstreams (e.g., opencode.ai gateway) require `X-Api-Key`, resulting in 401 auth failures. Fixed to send both `X-Api-Key` and `Anthropic-Version` headers
- **New test coverage**: Added `TestChatCompletionsEndpointUsesAnthropicAuth` to verify correct auth header behavior on chat/completions path

---

# 历史版本 / Previous Releases

## v2.0.3

### 修复 / Fixes
- **费用估算双重计费（严重）**：`EstimateCost` 对缓存读取的 tokens 既按全价计费又按缓存价计费，导致有缓存的请求费用虚高约 2-23 倍
- **`extractUsageFromAnthropicStream` 缺字段**：只解析 `message_delta` 事件，遗漏了 `message_start` 中的 `input_tokens` / cache 字段
- **重试导致请求次数虚高**：每次重试失败都写入历史记录，导致 1 次用户请求在统计中最多被计为 6 次
- **`modelBreakdown` 缓存命中率误含写入**：命中率分子使用了 `CacheRead + CacheCreation`，修复后只使用 `CacheRead`
- **流式 `message_delta` 缺 `input_tokens`**：OpenAI → Anthropic 协议转换的合成 `message_delta` 未包含 `input_tokens`
- **流量界面选择"今日"时间窗口错误**：使用 `time.Now()` 导致显示为近 24h 而非当日数据

## v2.0.2 — 流量监控 / 额度看板 / 客户端集成 / 多巴胺配色

## v2.0.1 — ccswitch / claude-desktop-env CLI 增强

## v2.0.0 — 原生双语控制面板发布 / Premium Bilingual Desktop Control Panel

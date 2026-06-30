package pricing

import "strings"

// ModelPrice 单个模型的定价信息
type ModelPrice struct {
	InputPricePer1M   float64 // Input $/1M tokens
	OutputPricePer1M  float64 // Output $/1M tokens
	CacheReadPricePer1M  float64 // Cache read $/1M tokens
	CacheWritePricePer1M float64 // Cache write $/1M tokens (0 if N/A)
	Provider          string  // 模型提供商
}

// SpendingLimit OpenCode Go 额度限制
type SpendingLimit struct {
	Label string  // 限制名称
	Limit float64 // 金额上限 ($)
}

// PlanUsage 套餐使用情况（用于前端展示）
type PlanUsage struct {
	TotalCost   float64 `json:"total_cost"`   // 当前总花费
	FiveHour    LimitUsage `json:"five_hour"`
	Weekly      LimitUsage `json:"weekly"`
	Monthly     LimitUsage `json:"monthly"`
}

type LimitUsage struct {
	Limit  float64 `json:"limit"`
	Used   float64 `json:"used"`
	Remain float64 `json:"remain"`
	Pct    float64 `json:"pct"`
}

// 已知模型定价表（来源: OpenCode Go 官方定价）
// 模型名使用小写 + 连字符格式，匹配 API 返回的 model 字段
var ModelPrices = map[string]ModelPrice{
	// GLM (Zhipu)
	"glm-5.2":             {InputPricePer1M: 1.60, OutputPricePer1M: 5.00, CacheReadPricePer1M: 0.30, CacheWritePricePer1M: 0, Provider: "Zhipu"},
	"glm-5.1":             {InputPricePer1M: 1.40, OutputPricePer1M: 4.40, CacheReadPricePer1M: 0.26, CacheWritePricePer1M: 0, Provider: "Zhipu"},
	"glm-5":               {InputPricePer1M: 1.00, OutputPricePer1M: 3.20, CacheReadPricePer1M: 0.20, CacheWritePricePer1M: 0, Provider: "Zhipu"},

	// Kimi (Moonshot)
	"kimi-k2.7-code":      {InputPricePer1M: 1.20, OutputPricePer1M: 4.80, CacheReadPricePer1M: 0.20, CacheWritePricePer1M: 0, Provider: "Moonshot"},
	"kimi-k2.7":           {InputPricePer1M: 1.20, OutputPricePer1M: 4.80, CacheReadPricePer1M: 0.20, CacheWritePricePer1M: 0, Provider: "Moonshot"},
	"kimi-k2.6":           {InputPricePer1M: 0.95, OutputPricePer1M: 4.00, CacheReadPricePer1M: 0.16, CacheWritePricePer1M: 0, Provider: "Moonshot"},
	"kimi-k2.5":           {InputPricePer1M: 0.60, OutputPricePer1M: 3.00, CacheReadPricePer1M: 0.10, CacheWritePricePer1M: 0, Provider: "Moonshot"},

	// MiMo
	"mimo-v2.5":           {InputPricePer1M: 0.14, OutputPricePer1M: 0.28, CacheReadPricePer1M: 0.0028, CacheWritePricePer1M: 0, Provider: "MiMo"},
	"mimo-v2.5-pro":       {InputPricePer1M: 1.74, OutputPricePer1M: 3.48, CacheReadPricePer1M: 0.0145, CacheWritePricePer1M: 0, Provider: "MiMo"},

	// MiniMax
	"minimax-m3":          {InputPricePer1M: 0.60, OutputPricePer1M: 2.40, CacheReadPricePer1M: 0.12, CacheWritePricePer1M: 0.75, Provider: "MiniMax"},
	"minimax-m2.7":        {InputPricePer1M: 0.30, OutputPricePer1M: 1.20, CacheReadPricePer1M: 0.06, CacheWritePricePer1M: 0.375, Provider: "MiniMax"},
	"minimax-m2.5":        {InputPricePer1M: 0.30, OutputPricePer1M: 1.20, CacheReadPricePer1M: 0.06, CacheWritePricePer1M: 0.375, Provider: "MiniMax"},

	// Qwen (Alibaba)
	"qwen3.7-max":         {InputPricePer1M: 2.50, OutputPricePer1M: 7.50, CacheReadPricePer1M: 0.50, CacheWritePricePer1M: 3.125, Provider: "Alibaba"},
	"qwen3.7-plus":        {InputPricePer1M: 1.00, OutputPricePer1M: 4.50, CacheReadPricePer1M: 0.15, CacheWritePricePer1M: 1.00, Provider: "Alibaba"},
	"qwen3.6-plus":        {InputPricePer1M: 0.50, OutputPricePer1M: 3.00, CacheReadPricePer1M: 0.05, CacheWritePricePer1M: 0.625, Provider: "Alibaba"},

	// DeepSeek
	"deepseek-v4-pro":     {InputPricePer1M: 1.74, OutputPricePer1M: 3.48, CacheReadPricePer1M: 0.0145, CacheWritePricePer1M: 0, Provider: "DeepSeek"},
	"deepseek-v4-flash":   {InputPricePer1M: 0.14, OutputPricePer1M: 0.28, CacheReadPricePer1M: 0.0028, CacheWritePricePer1M: 0, Provider: "DeepSeek"},

	// 默认（未知模型取 DeepSeek V4 Flash 价格作为合理估算）
	"default":             {InputPricePer1M: 0.14, OutputPricePer1M: 0.28, CacheReadPricePer1M: 0.0028, CacheWritePricePer1M: 0, Provider: "Unknown"},
}

// SpendingLimits OpenCode Go 套餐额度限制
var SpendingLimits = []SpendingLimit{
	{Label: "5小时限制", Limit: 12.00},
	{Label: "每周限制", Limit: 30.00},
	{Label: "每月限制", Limit: 60.00},
}

// ModelRequestQuota 官方给定的每个模型月度请求配额（OpenCode Go 套餐）
// 来源: OpenCode Go 官方定价页 - 每月可发送请求数
// 注意: 这些是硬编码的官方配额，不要计算，直接使用
var ModelRequestQuota = map[string]int{
	// Big Pickle 和免费模型（共享配额）
	"big-pickle":          200,
	"free":                200,

	// GLM (Zhipu)
	"glm-5.2":             880,
	"glm-5.1":             880,
	"glm-5":               880,

	// Qwen (Alibaba)
	"qwen3.7-max":         950,
	"qwen3.7-plus":        4300,
	"qwen3.6-plus":        4300,
	"qwen3.5-plus":        4300,

	// Kimi (Moonshot)
	"kimi-k2.7-code":      1150,
	"kimi-k2.7":           1150,
	"kimi-k2.6":           1150,
	"kimi-k2.5":           1150,

	// MiMo
	"mimo-v2.5-pro":       3250,
	"mimo-v2.5":           30100,

	// DeepSeek
	"deepseek-v4-pro":     3450,
	"deepseek-v4-flash":   31650,

	// MiniMax (3x usage)
	"minimax-m3":          9600,
	"minimax-m2.7":        9600,
	"minimax-m2.5":        9600,
}

// GetModelQuota 返回模型的官方月度请求配额
// 大小写无关匹配；未配置的模型返回 0
func GetModelQuota(model string) int {
	return ModelRequestQuota[normalizeModelName(model)]
}

// normalizeModelName 将模型名归一化为小写 + 连字符格式，用于大小写无关匹配
// 例如 "GLM-5.2" → "glm-5.2", "Kimi K2.7 Code" → "kimi k2.7 code"
func normalizeModelName(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}

// EstimateCost 估算单次请求的花费（含缓存）
func EstimateCost(model string, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int) float64 {
	price, ok := ModelPrices[normalizeModelName(model)]
	if !ok {
		price = ModelPrices["default"]
	}
	// input_tokens 已包含 cache_read_tokens（后者是子集），
	// 缓存读取部分应按缓存价而非全价计费，故从 input 中减去
	nonCacheInput := inputTokens - cacheReadTokens
	if nonCacheInput < 0 {
		nonCacheInput = 0
	}
	inputCost := (float64(nonCacheInput) / 1_000_000) * price.InputPricePer1M
	outputCost := (float64(outputTokens) / 1_000_000) * price.OutputPricePer1M
	cacheReadCost := (float64(cacheReadTokens) / 1_000_000) * price.CacheReadPricePer1M
	cacheWriteCost := (float64(cacheWriteTokens) / 1_000_000) * price.CacheWritePricePer1M
	return inputCost + outputCost + cacheReadCost + cacheWriteCost
}

// EstimateSpendingUsage 根据总花费计算各额度限制的使用情况
func EstimateSpendingUsage(totalCost float64) PlanUsage {
	usage := PlanUsage{TotalCost: totalCost}

	for _, sl := range SpendingLimits {
		pct := (totalCost / sl.Limit) * 100
		if pct > 100 {
			pct = 100
		}
		lu := LimitUsage{
			Limit:  sl.Limit,
			Used:   totalCost,
			Remain: sl.Limit - totalCost,
			Pct:    pct,
		}
		if lu.Remain < 0 {
			lu.Remain = 0
		}
		switch sl.Label {
		case "5小时限制":
			usage.FiveHour = lu
		case "每周限制":
			usage.Weekly = lu
		case "每月限制":
			usage.Monthly = lu
		}
	}
	return usage
}

// GetProvider 返回模型对应的提供商名称
func GetProvider(model string) string {
	if price, ok := ModelPrices[normalizeModelName(model)]; ok {
		return price.Provider
	}
	return "Unknown"
}

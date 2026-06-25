package quota

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const openCodeGoBaseURL = "https://opencode.ai"

// QuotaUsage represents quota usage for a single time dimension.
type QuotaUsage struct {
	Status       string `json:"status"`        // "active" or "unlimited"
	UsagePercent int    `json:"usage_percent"`  // 0–100
	ResetInSec   int    `json:"reset_in_sec"`
	ResetDisplay string `json:"reset_display"`  // compact duration: "2h", "30m", "5d"
}

// QuotaData holds rolling / weekly / monthly quota info.
type QuotaData struct {
	Rolling   QuotaUsage  `json:"rolling"`
	Weekly    QuotaUsage  `json:"weekly"`
	Monthly   *QuotaUsage `json:"monthly,omitempty"`
	FetchedAt time.Time   `json:"fetched_at"`
}

// QuotaResult is the JSON API response envelope.
type QuotaResult struct {
	Success      bool       `json:"success"`
	ProviderName string     `json:"provider_name"`
	Data         *QuotaData `json:"data,omitempty"`
	Error        string     `json:"error,omitempty"`
}

// pageGoUsage mirrors the JSON shape embedded in the /workspace/{id}/go page.
type pageGoUsage struct {
	Rolling *windowData `json:"rollingUsage"`
	Weekly  *windowData `json:"weeklyUsage"`
	Monthly *windowData `json:"monthlyUsage"`
}
type windowData struct {
	Status       *string  `json:"status"`
	UsagePercent *float64 `json:"usagePercent"`
	UsedPercent  *float64 `json:"usedPercent"`
	ResetInSec   *float64 `json:"resetInSec"`
}

// FetchOpenCodeGoQuota retrieves current Go plan quota from the
// /workspace/{id}/go page. Tries JSON first, falls back to regex.
func FetchOpenCodeGoQuota(cookie, workspaceID string) (*QuotaData, error) {
	if cookie == "" {
		return nil, fmt.Errorf("OpenCode Go auth cookie not configured (see quota_cookie in profile config)")
	}
	if workspaceID == "" {
		return nil, fmt.Errorf("OpenCode Go workspace ID not configured (see quota_workspace_id in profile config)")
	}

	pageURL := fmt.Sprintf("%s/workspace/%s/go", openCodeGoBaseURL, workspaceID)
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("accept", "*/*")
	req.Header.Set("cookie", cookie)
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch quota: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("quota page returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return parseGoUsage(string(body))
}

// CredentialsFromEnv reads OpenCode Go credentials from environment variables.
func CredentialsFromEnv() (cookie, workspaceID string) {
	cookie = os.Getenv("OPENCODE_GO_AUTH_COOKIE")
	workspaceID = os.Getenv("OPENCODE_GO_WORKSPACE_ID")
	return
}

// parseGoUsage tries JSON first, then falls back to regex.
func parseGoUsage(text string) (*QuotaData, error) {
	data, err := parseGoUsageJSON(text)
	if err == nil && data != nil {
		return data, nil
	}
	return parseGoUsageRegex(text)
}

// parseGoUsageJSON attempts to unmarshal the page body as JSON.
func parseGoUsageJSON(text string) (*QuotaData, error) {
	var page pageGoUsage
	if err := json.Unmarshal([]byte(text), &page); err != nil {
		return nil, err
	}
	if page.Rolling == nil || page.Weekly == nil {
		return nil, fmt.Errorf("incomplete JSON: missing rolling or weekly data")
	}
	rolling := buildUsage(
		toStatus(page.Rolling), toPct(page.Rolling), toSec(page.Rolling))
	weekly := buildUsage(
		toStatus(page.Weekly), toPct(page.Weekly), toSec(page.Weekly))
	var monthly *QuotaUsage
	if page.Monthly != nil {
		m := buildUsage(toStatus(page.Monthly), toPct(page.Monthly), toSec(page.Monthly))
		if m.Status != "unlimited" {
			monthly = &m
		}
	}
	return &QuotaData{
		Rolling:   rolling,
		Weekly:    weekly,
		Monthly:   monthly,
		FetchedAt: time.Now(),
	}, nil
}

// parseGoUsageRegex extracts window data from text/javascript responses (fallback).
// Each window key + its fields is matched in a single regex for accuracy.
func parseGoUsageRegex(text string) (*QuotaData, error) {
	type windowMatch struct {
		pct     int
		resetMs int
	}
	extract := func(key string) (windowMatch, bool) {
		pctRe := regexp.MustCompile(key + `[^}]*?usagePercent"?\s*[:=]\s*([0-9]+(?:\.[0-9]+)?)`)
		pm := pctRe.FindStringSubmatch(text)
		if pm == nil {
			return windowMatch{}, false
		}
		resetRe := regexp.MustCompile(key + `[^}]*?resetInSec"?\s*[:=]\s*([0-9]+)`)
		rm := resetRe.FindStringSubmatch(text)
		resetSec := 0
		if rm != nil {
			resetSec, _ = strconv.Atoi(rm[1])
		}
		return windowMatch{
			pct:     clampPct(int(parseFloat(pm[1]))),
			resetMs: resetSec,
		}, true
	}

	rolling, rok := extract("rollingUsage")
	weekly, wok := extract("weeklyUsage")
	if !rok || !wok {
		return nil, fmt.Errorf("failed to parse usagePercent from page response")
	}

	result := &QuotaData{
		Rolling:   buildUsage("", rolling.pct, rolling.resetMs),
		Weekly:    buildUsage("", weekly.pct, weekly.resetMs),
		FetchedAt: time.Now(),
	}
	if monthly, mok := extract("monthlyUsage"); mok {
		m := buildUsage("", monthly.pct, monthly.resetMs)
		if m.Status != "unlimited" {
			result.Monthly = &m
		}
	}
	return result, nil
}

// --- helpers ----------------------------------------------------------------

func toStatus(w *windowData) string {
	if w.Status != nil {
		return *w.Status
	}
	return ""
}

func toPct(w *windowData) int {
	if w.UsagePercent != nil {
		return clampPct(int(*w.UsagePercent))
	}
	if w.UsedPercent != nil {
		return clampPct(int(*w.UsedPercent))
	}
	return 0
}

func toSec(w *windowData) int {
	if w.ResetInSec != nil {
		return int(*w.ResetInSec)
	}
	return 0
}

func buildUsage(status string, pct, resetSec int) QuotaUsage {
	if status == "" {
		status = "active"
	}
	return QuotaUsage{
		Status:       status,
		UsagePercent: pct,
		ResetInSec:   resetSec,
		ResetDisplay: formatDurationCompact(resetSec),
	}
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func clampPct(pct int) int {
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// formatDurationCompact formats a duration in seconds as a compact human-readable string.
func formatDurationCompact(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	return fmt.Sprintf("%dd", seconds/86400)
}

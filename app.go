package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/ethan-blue/open-code-go-tools/internal/config"
	"github.com/ethan-blue/open-code-go-tools/internal/proxy"
	"github.com/getlantern/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	srv        *proxy.Server
	cancelFunc context.CancelFunc
}

// NewApp creates a new App struct instance
func NewApp() *App {
	return &App{}
}

//go:embed build/appicon.png
var appIconPng []byte

//go:embed build/windows/icon.ico
var appIconIco []byte

func (a *App) setupSystray() {
	go systray.Run(func() {
		if runtime.GOOS == "windows" {
			systray.SetIcon(appIconIco)
		} else {
			systray.SetIcon(appIconPng)
		}
		systray.SetTitle("ocgt")
		systray.SetTooltip("ocgt 控制面板 - Claude API 本地代理服务")

		mShow := systray.AddMenuItem("显示控制面板", "显示主窗口")
		mHide := systray.AddMenuItem("隐藏控制面板", "隐藏主窗口")
		systray.AddSeparator()
		mQuit := systray.AddMenuItem("退出程序", "彻底退出代理服务")

		// Use context for proper goroutine cleanup
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()
			for {
				select {
				case <-ctx.Done():
					return
				case <-mShow.ClickedCh:
					a.showMainWindow()
				case <-mHide.ClickedCh:
					if a.ctx != nil {
						wailsruntime.WindowHide(a.ctx)
					}
				case <-mQuit.ClickedCh:
					systray.Quit()
					if a.ctx != nil {
						wailsruntime.Quit(a.ctx)
					}
					return
				}
			}
		}()
	}, func() {
		// Cleanup
	})
}

func (a *App) showMainWindow() {
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.WindowCenter(a.ctx)
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Delay the system tray setup slightly (500ms) to allow Wails WebView2 message loop
	// to register first and avoid thread contention/focus stealing on Windows startup.
	go func() {
		time.Sleep(500 * time.Millisecond)
		a.setupSystray()
	}()

	// Start Go proxy server in the background!
	go func() {
		// 1. Auto-init config if it doesn't exist
		defaultPath, err := config.DefaultPath()
		if err == nil {
			if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
				_, _ = config.WriteExample("", false)
			}
		}

		// 2. Load config
		cfg, err := config.Load("")
		if err != nil {
			log.Println("[GUI proxy] config load error:", err)
			return
		}

		// 3. Create server
		srv, err := proxy.New(cfg)
		if err != nil {
			log.Println("[GUI proxy] server creation error:", err)
			return
		}
		srv.SetConfigPath(defaultPath)
		a.srv = srv

		// 4. Listen and Serve with cancellation context
		proxyCtx, cancel := context.WithCancel(context.Background())
		a.cancelFunc = cancel

		log.Println("[GUI proxy] starting background proxy server on http://" + cfg.Listen)
		if err := srv.ListenAndServe(proxyCtx); err != nil {
			log.Println("[GUI proxy] server stopped:", err)
		}
	}()
}

// domReady is called when the frontend DOM is fully loaded and ready.
func (a *App) domReady(ctx context.Context) {
	a.ctx = ctx
	// Force the main window to be shown, unminimized, centered and focused on startup
	a.showMainWindow()
}

// shutdown is called when the app closes
func (a *App) shutdown(ctx context.Context) {
	if a.cancelFunc != nil {
		log.Println("[GUI proxy] shutting down background proxy server...")
		a.cancelFunc()
	}
}

// GetListenAddress returns the actual proxy listen address dynamically
func (a *App) GetListenAddress() string {
	if a.srv != nil {
		return a.srv.ListenAddress()
	}
	// Try loading config to get the address if server is not fully initialized yet
	cfg, err := config.Load("")
	if err == nil && cfg.Listen != "" {
		return cfg.Listen
	}
	return "127.0.0.1:8787" // default fallback
}

// SaveProfileConfig saves API key, model aliases, timeout, and thinking settings.
func (a *App) SaveProfileConfig(profileName, apiKey, defaultModel, sonnetAlias, haikuAlias, opusAlias, timeoutSeconds, thinkingBudgetTokens string) string {
	// 1. Resolve path
	path, err := config.DefaultPath()
	if err != nil {
		return "resolve path error: " + err.Error()
	}

	// 2. Load config
	cfg, err := config.Load(path)
	if err != nil {
		return "load error: " + err.Error()
	}

	// 3. Find and update profile
	p, ok := cfg.Profiles[profileName]
	if !ok {
		return "profile not found: " + profileName
	}

	if apiKey != "" {
		p.APIKey = apiKey
	}
	p.DefaultModel = defaultModel
	if p.ModelAliases == nil {
		p.ModelAliases = make(map[string]string)
	}
	p.ModelAliases["sonnet"] = sonnetAlias
	p.ModelAliases["haiku"] = haikuAlias
	p.ModelAliases["opus"] = opusAlias
	cfg.Profiles[profileName] = p
	if timeoutSeconds != "" {
		timeout, err := strconv.Atoi(timeoutSeconds)
		if err != nil {
			return "request timeout must be a number of seconds"
		}
		cfg.RequestTimeoutSeconds = timeout
	}
	if thinkingBudgetTokens != "" {
		budget, err := strconv.Atoi(thinkingBudgetTokens)
		if err != nil {
			return "thinking budget must be a number of tokens"
		}
		cfg.MaxThinkingBudgetTokens = budget
	}
	if err := cfg.Validate(); err != nil {
		return "validation error: " + err.Error()
	}

	// 4. Save config
	if err := cfg.Save(path); err != nil {
		return "save error: " + err.Error()
	}

	// 5. Update server config in-memory if running
	if a.srv != nil {
		a.srv.ApplyConfig(cfg)
	}

	return "success"
}

// InstallClaudeUserEnv persists Claude Code environment variables for new shells.
func (a *App) InstallClaudeUserEnv() string {
	listenAddr := a.GetListenAddress()
	activeProfile := "opencode-go"
	defaultModel := "kimi-k2.6"
	thinkingBudget := config.DefaultMaxThinkingBudgetTokens

	path, err := config.DefaultPath()
	if err == nil {
		cfg, err := config.Load(path)
		if err == nil {
			activeProfile = cfg.ActiveProfile
			thinkingBudget = cfg.ThinkingBudgetTokens()
			if p, ok := cfg.Profiles[activeProfile]; ok && p.DefaultModel != "" {
				defaultModel = p.DefaultModel
			}
		}
	}

	env := map[string]string{
		"ANTHROPIC_BASE_URL":       "http://" + listenAddr,
		"ANTHROPIC_API_KEY":        "ocgt-local-proxy",
		"ANTHROPIC_CUSTOM_HEADERS": "X-Ocgt-Profile: " + activeProfile,
		"ANTHROPIC_MODEL":          defaultModel,
		"OCGT_PROFILE":             activeProfile,
	}
	applyClaudeThinkingEnv(env, thinkingBudget)

	for _, name := range legacyClaudeEnvNames() {
		if err := unsetUserEnvironment(name); err != nil {
			return "unset " + name + " error: " + err.Error()
		}
	}
	for name, value := range env {
		if err := setUserEnvironment(name, value); err != nil {
			return "set " + name + " error: " + err.Error()
		}
	}
	if err := syncClaudeSettings(env, defaultModel); err != nil {
		return "sync Claude settings error: " + err.Error()
	}
	return "success"
}

// sanitizeEnvValue validates that a value is safe to pass as an environment variable.
// It only allows alphanumeric characters, dash, underscore, dot, colon, slash, and space.
func sanitizeEnvValue(value, name string) error {
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == ':' || r == '/' || r == ' ':
		default:
			return fmt.Errorf("invalid character %q in %s", r, name)
		}
	}
	return nil
}

// LaunchClaudeTerminal spawns a new terminal window preconfigured with the Claude Code proxy environment
func (a *App) LaunchClaudeTerminal(shell string) string {
	listenAddr := a.GetListenAddress()
	activeProfile := "opencode-go"
	defaultModel := "kimi-k2.6"
	thinkingBudget := config.DefaultMaxThinkingBudgetTokens

	// Try loading from config to get the latest
	path, err := config.DefaultPath()
	if err == nil {
		cfg, err := config.Load(path)
		if err == nil {
			activeProfile = cfg.ActiveProfile
			thinkingBudget = cfg.ThinkingBudgetTokens()
			if p, ok := cfg.Profiles[activeProfile]; ok {
				if p.DefaultModel != "" {
					defaultModel = p.DefaultModel
				}
			}
		}
	}

	// Validate inputs to prevent command injection
	if err := sanitizeEnvValue(activeProfile, "profile name"); err != nil {
		return "invalid profile name: " + err.Error()
	}
	if err := sanitizeEnvValue(defaultModel, "model name"); err != nil {
		return "invalid model name: " + err.Error()
	}
	thinkingEnv := map[string]string{}
	applyClaudeThinkingEnv(thinkingEnv, thinkingBudget)
	thinkingTokenValue := thinkingEnv["MAX_THINKING_TOKENS"]
	disableThinking := thinkingEnv["CLAUDE_CODE_DISABLE_THINKING"] == "1"

	baseURL := "http://" + listenAddr

	switch runtime.GOOS {
	case "windows":
		// Use environment variable passing instead of string interpolation for security
		env := []string{
			fmt.Sprintf("ANTHROPIC_BASE_URL=%s", baseURL),
			"ANTHROPIC_API_KEY=ocgt-local-proxy",
			fmt.Sprintf("ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile:%s", activeProfile),
			fmt.Sprintf("ANTHROPIC_MODEL=%s", defaultModel),
			fmt.Sprintf("MAX_THINKING_TOKENS=%s", thinkingTokenValue),
		}
		if disableThinking {
			env = append(env, "CLAUDE_CODE_DISABLE_THINKING=1")
		}

		if shell == "cmd" {
			// Launch CMD terminal - use cmd.Env for secure env passing
			cmd := exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/k",
				"echo =========================================================&& "+
					"echo  [ocgt] Claude Code 代理终端已成功拉起！&& "+
					"echo  当前代理: "+baseURL+"&& "+
					"echo  当前模型: "+defaultModel+"&& "+
					"echo  请在下方直接输入: claude&& "+
					"echo =========================================================&& echo.")
			cmd.Env = mergedClaudeProcessEnv(env, disableThinking)
			if err := cmd.Run(); err != nil {
				return "launch cmd error: " + err.Error()
			}
		} else {
			// Launch PowerShell terminal - use -EncodedCommand for security
			psScript := fmt.Sprintf(
				"Remove-Item Env:ANTHROPIC_AUTH_TOKEN -ErrorAction SilentlyContinue; "+
					"$env:ANTHROPIC_BASE_URL='%s'; "+
					"$env:ANTHROPIC_API_KEY='ocgt-local-proxy'; "+
					"$env:ANTHROPIC_CUSTOM_HEADERS='X-Ocgt-Profile: %s'; "+
					"$env:ANTHROPIC_MODEL='%s'; "+
					"$env:MAX_THINKING_TOKENS='%s'; "+
					"%s"+
					"Clear-Host; "+
					"Write-Host '=========================================================' -ForegroundColor Cyan; "+
					"Write-Host ' [ocgt] Claude Code 代理终端已成功拉起！' -ForegroundColor Green; "+
					"Write-Host ' 当前代理: %s' -ForegroundColor Gray; "+
					"Write-Host ' 当前模型: %s' -ForegroundColor Gray; "+
					"Write-Host ' 请在下方直接输入: claude' -ForegroundColor Green; "+
					"Write-Host '=========================================================' -ForegroundColor Cyan; "+
					"Write-Host ''",
				baseURL, activeProfile, defaultModel, thinkingTokenValue, powershellThinkingDisableScript(disableThinking), baseURL, defaultModel)
			cmd := exec.Command("powershell.exe", "-NoExit", "-Command", psScript)
			cmd.Env = mergedClaudeProcessEnv(env, disableThinking)
			if err := cmd.Start(); err != nil {
				return "launch powershell error: " + err.Error()
			}
		}
		return "success"
	case "darwin":
		// MacOS support (Terminal.app) - use env vars via export commands
		// The values are already validated above
		script := fmt.Sprintf(
			`tell application "Terminal" to do script "unset ANTHROPIC_AUTH_TOKEN && export ANTHROPIC_BASE_URL='%s' && export ANTHROPIC_API_KEY='ocgt-local-proxy' && export ANTHROPIC_CUSTOM_HEADERS='X-Ocgt-Profile: %s' && export ANTHROPIC_MODEL='%s' && export MAX_THINKING_TOKENS='%s' && %sclear && echo '=========================================================' && echo ' [ocgt] Claude Code 代理终端已成功拉起！' && echo ' 当前代理: %s' && echo ' 当前模型: %s' && echo ' 请在下方直接输入: claude' && echo '=========================================================' && echo ''"`,
			baseURL, activeProfile, defaultModel, thinkingTokenValue, shellThinkingDisableScript(disableThinking), baseURL, defaultModel)
		cmd := exec.Command("osascript", "-e", script)
		if err := cmd.Run(); err != nil {
			return "launch terminal error: " + err.Error()
		}
		return "success"
	default:
		return "unsupported operating system for automatic terminal launch"
	}
}

func unsetUserEnvironment(name string) error {
	if err := os.Unsetenv(name); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return unsetWindowsUserEnvironment(name)
	case "darwin":
		return nil
	default:
		return nil
	}
}

func legacyClaudeEnvNames() []string {
	return []string{
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_DEFAULT_SONNET_MODEL_NAME",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL_NAME",
		"ANTHROPIC_DEFAULT_OPUS_MODEL_NAME",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS",
		"CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY",
		"CLAUDE_CODE_DISABLE_THINKING",
	}
}

func applyClaudeThinkingEnv(env map[string]string, budgetTokens int) {
	if budgetTokens < 0 {
		env["MAX_THINKING_TOKENS"] = "0"
		env["CLAUDE_CODE_DISABLE_THINKING"] = "1"
		return
	}
	if budgetTokens == 0 {
		budgetTokens = config.DefaultMaxThinkingBudgetTokens
	}
	env["MAX_THINKING_TOKENS"] = strconv.Itoa(budgetTokens)
}

func mergedClaudeProcessEnv(overrides []string, disableThinking bool) []string {
	drop := map[string]bool{
		"ANTHROPIC_BASE_URL":           true,
		"ANTHROPIC_API_KEY":            true,
		"ANTHROPIC_CUSTOM_HEADERS":     true,
		"ANTHROPIC_MODEL":              true,
		"MAX_THINKING_TOKENS":          true,
		"CLAUDE_CODE_DISABLE_THINKING": !disableThinking,
	}
	out := make([]string, 0, len(os.Environ())+len(overrides))
	for _, item := range os.Environ() {
		name, _, found := strings.Cut(item, "=")
		if found && drop[name] {
			continue
		}
		out = append(out, item)
	}
	return append(out, overrides...)
}

func powershellThinkingDisableScript(disabled bool) string {
	if disabled {
		return "$env:CLAUDE_CODE_DISABLE_THINKING='1'; "
	}
	return "Remove-Item Env:CLAUDE_CODE_DISABLE_THINKING -ErrorAction SilentlyContinue; "
}

func shellThinkingDisableScript(disabled bool) string {
	if disabled {
		return "export CLAUDE_CODE_DISABLE_THINKING='1' && "
	}
	return "unset CLAUDE_CODE_DISABLE_THINKING && "
}

// OpenConfigLocation opens the directory containing the config file
func (a *App) OpenConfigLocation() string {
	path, err := config.DefaultPath()
	if err != nil {
		return "resolve path error: " + err.Error()
	}
	dir := filepath.Dir(path)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		return "unsupported operating system"
	}

	if err := cmd.Start(); err != nil {
		return "open error: " + err.Error()
	}
	return "success"
}

func setUserEnvironment(name, value string) error {
	if err := os.Setenv(name, value); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "windows":
		return setWindowsUserEnvironment(name, value)
	case "darwin":
		return nil
	default:
		return nil
	}
}

func syncClaudeSettings(env map[string]string, defaultModel string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	settings := map[string]any{}
	if data, err := os.ReadFile(settingsPath); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	envMap, _ := settings["env"].(map[string]any)
	if envMap == nil {
		envMap = map[string]any{}
	}
	for _, name := range legacyClaudeEnvNames() {
		delete(envMap, name)
	}

	for key, value := range env {
		envMap[key] = value
	}
	settings["env"] = envMap

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o700); err != nil {
		return err
	}
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, append(out, '\n'), 0o600)
}

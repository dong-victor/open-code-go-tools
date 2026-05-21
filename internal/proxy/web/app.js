// 全局状态变量
let systemStatus = null;
let currentShell = 'powershell';

// DOM 元素引用
const elListen = document.getElementById('status-listen');
const elUpstream = document.getElementById('status-upstream');
const elProfile = document.getElementById('status-profile');
const elModel = document.getElementById('status-model');
const elConfigPath = document.getElementById('status-config-path');
const elTimeout = document.getElementById('status-timeout');
const selectProfile = document.getElementById('profile-select');
const inputApiKey = document.getElementById('api-key-input');
const btnSaveKey = document.getElementById('save-key-btn');
const btnToggleVisibility = document.getElementById('toggle-key-visibility');
const shellTabs = document.getElementById('shell-tabs');
const codeEnv = document.getElementById('env-code-block');
const codeCCSwitch = document.getElementById('ccswitch-code-block');
const btnCopyEnv = document.getElementById('copy-env-btn');
const btnCopyCCSwitch = document.getElementById('copy-ccswitch-btn');
const tbodyHistory = document.getElementById('history-tbody');

// 初始化入口
document.addEventListener('DOMContentLoaded', () => {
    loadStatus();
    loadProfiles();
    loadHistory();
    
    // 定时轮询系统状态与流量历史 (每 2.5 秒)
    setInterval(() => {
        loadHistory();
    }, 2500);

    // 绑定交互事件
    setupEventHandlers();
});

// 加载系统基本状态
async function loadStatus() {
    try {
        const response = await fetch('/ocgt/api/status');
        if (!response.ok) throw new Error('Failed to load status');
        systemStatus = await response.json();
        
        // 渲染文本状态
        elListen.textContent = systemStatus.listen;
        elUpstream.textContent = systemStatus.upstream;
        elProfile.textContent = systemStatus.active_profile;
        elModel.textContent = systemStatus.default_model || '未设定';
        elConfigPath.textContent = systemStatus.config_path;
        if (elTimeout) {
            elTimeout.textContent = `${systemStatus.request_timeout_seconds || 300}s`;
        }
        
        // 更新命令模板
        renderEnvCode();
        renderCCSwitchCode();
    } catch (err) {
        console.error('Error fetching system status:', err);
    }
}

// 加载配置里的 Profiles 列表并更新下拉框
async function loadProfiles() {
    try {
        const response = await fetch('/ocgt/api/profiles');
        if (!response.ok) throw new Error('Failed to load profiles');
        const data = await response.json();
        
        selectProfile.innerHTML = '';
        Object.keys(data.profiles).forEach(pName => {
            const option = document.createElement('option');
            option.value = pName;
            option.textContent = pName;
            if (pName === data.active_profile) {
                option.selected = true;
            }
            selectProfile.appendChild(option);
        });
    } catch (err) {
        console.error('Error loading profiles:', err);
    }
}

// 异步加载流量日志
async function loadHistory() {
    try {
        const response = await fetch('/ocgt/api/history');
        if (!response.ok) throw new Error('Failed to load history');
        const history = await response.json();
        
        renderHistoryTable(history);
    } catch (err) {
        console.error('Error loading request history:', err);
    }
}

// 绑定事件处理器
function setupEventHandlers() {
    // 监听 Profile 下拉菜单选择事件
    selectProfile.addEventListener('change', async (e) => {
        const newProfile = e.target.value;
        try {
            const resp = await fetch('/ocgt/api/profiles/active', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ profile: newProfile })
            });
            if (resp.ok) {
                // 刷新系统状态
                await loadStatus();
                // 闪烁提示切换成功
                flashElement(selectProfile, 'rgba(16, 185, 129, 0.2)');
            }
        } catch (err) {
            console.error('Failed to change active profile:', err);
        }
    });

    // 密钥明密文切换
    btnToggleVisibility.addEventListener('click', () => {
        const isPassword = inputApiKey.type === 'password';
        inputApiKey.type = isPassword ? 'text' : 'password';
        btnToggleVisibility.classList.toggle('visible');
    });

    // API Key 保存处理器
    btnSaveKey.addEventListener('click', async () => {
        const key = inputApiKey.value.trim();
        if (!key) {
            alert('请输入合法的 API Key。');
            return;
        }

        const activeProfile = selectProfile.value;
        setButtonState(btnSaveKey, 'saving');
        
        try {
            const resp = await fetch('/ocgt/api/key', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    profile: activeProfile,
                    api_key: key
                })
            });
            
            if (resp.ok) {
                setButtonState(btnSaveKey, 'success');
                inputApiKey.value = ''; // 清空密码框
                setTimeout(() => {
                    setButtonState(btnSaveKey, 'idle');
                }, 1500);
            } else {
                setButtonState(btnSaveKey, 'idle');
                alert('保存失败，请检查后端运行日志。');
            }
        } catch (err) {
            console.error('Failed to save API key:', err);
            setButtonState(btnSaveKey, 'idle');
        }
    });

    // 切换环境导入终端选择器
    shellTabs.addEventListener('click', (e) => {
        const btn = e.target.closest('.tab-btn');
        if (!btn) return;
        
        shellTabs.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        currentShell = btn.dataset.shell;
        renderEnvCode();
    });

    // 复制环境命令
    btnCopyEnv.addEventListener('click', () => {
        copyTextToClipboard(codeEnv.innerText, btnCopyEnv);
    });

    // 复制 CC Switch JSON 片段
    btnCopyCCSwitch.addEventListener('click', () => {
        copyTextToClipboard(codeCCSwitch.innerText, btnCopyCCSwitch);
    });
}

// 动态渲染环境命令文本
function renderEnvCode() {
    if (!systemStatus) return;
    
    const listen = systemStatus.listen;
    const profile = systemStatus.active_profile;
    const model = systemStatus.default_model || 'kimi-k2.6';
    
    let template = '';
    
    switch (currentShell) {
        case 'bash':
            template = `export ANTHROPIC_BASE_URL="http://${listen}"\nexport ANTHROPIC_API_KEY="ocgt-local-proxy"\nexport ANTHROPIC_CUSTOM_HEADERS="X-Ocgt-Profile: ${profile}"\nexport CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS="1"\nexport CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY="1"\nexport ANTHROPIC_MODEL="${model}"`;
            codeEnv.className = 'language-bash';
            break;
        case 'cmd':
            template = `set ANTHROPIC_BASE_URL=http://${listen}\nset ANTHROPIC_API_KEY=ocgt-local-proxy\nset ANTHROPIC_CUSTOM_HEADERS=X-Ocgt-Profile:${profile}\nset CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1\nset CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY=1\nset ANTHROPIC_MODEL=${model}`;
            codeEnv.className = 'language-cmd';
            break;
        default: // powershell
            template = `$env:ANTHROPIC_BASE_URL = "http://${listen}"\n$env:ANTHROPIC_API_KEY = "ocgt-local-proxy"\n$env:ANTHROPIC_CUSTOM_HEADERS = "X-Ocgt-Profile: ${profile}"\n$env:CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS = "1"\n$env:CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY = "1"\n$env:ANTHROPIC_MODEL = "${model}"`;
            codeEnv.className = 'language-powershell';
            break;
    }
    
    codeEnv.innerText = template;
}

// 动态渲染 CC Switch JSON 片段
function renderCCSwitchCode() {
    if (!systemStatus) return;
    
    const listen = systemStatus.listen;
    const profile = systemStatus.active_profile;
    const model = systemStatus.default_model || 'kimi-k2.6';
    
    const cc = {
        name: `ocgt-${profile}`,
        type: "anthropic",
        baseURL: `http://${listen}`,
        apiKey: "ocgt-local-proxy",
        model: model,
        headers: {
            "X-Ocgt-Profile": profile
        }
    };
    
    codeCCSwitch.innerText = JSON.stringify(cc, null, 2);
}

// 流量表格渲染逻辑
function renderHistoryTable(logs) {
    if (!logs || logs.length === 0) {
        tbodyHistory.innerHTML = `
            <tr class="empty-state">
                <td colspan="7">
                    <div class="empty-message">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>
                        暂无代理流量记录，请在终端使用 Claude Code 发起对话...
                    </div>
                </td>
            </tr>
        `;
        return;
    }

    let rowsHTML = '';
    logs.forEach(log => {
        const timeStr = formatDate(new Date(log.time));
        const badgeClass = getStatusBadgeClass(log.status);
        
        rowsHTML += `
            <tr>
                <td class="font-mono">${timeStr}</td>
                <td><span style="font-weight: 700; color: #8b5cf6;">${log.method}</span></td>
                <td class="font-mono" style="color: var(--text-secondary);">${log.path}</td>
                <td class="font-mono" style="color: #60a5fa;">${log.model || '-'}</td>
                <td><span class="status-badge ${badgeClass}">${log.status}</span></td>
                <td class="font-mono" style="color: var(--success); font-weight: 600;">${log.duration}</td>
                <td class="error-cell" title="${escapeHtml(log.error || '')}">${escapeHtml(log.error || '-')}</td>
            </tr>
        `;
    });
    
    tbodyHistory.innerHTML = rowsHTML;
}

function escapeHtml(value) {
    return String(value)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}

// 格式化日期为 HH:mm:ss
function formatDate(date) {
    const pad = (n) => n.toString().padStart(2, '0');
    return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

// 状态码样式
function getStatusBadgeClass(status) {
    if (status >= 200 && status < 300) return 'status-2xx';
    if (status >= 400 && status < 500) return 'status-4xx';
    return 'status-5xx';
}

// 按钮状态切换机
function setButtonState(btn, state) {
    if (state === 'saving') {
        btn.disabled = true;
        btn.innerHTML = '<span>保存中...</span>';
        btn.style.opacity = '0.7';
    } else if (state === 'success') {
        btn.disabled = true;
        btn.className = 'btn btn-success';
        btn.innerHTML = '<span>保存成功！</span>';
        btn.style.opacity = '1';
    } else { // idle
        btn.disabled = false;
        btn.className = 'btn btn-primary';
        btn.innerHTML = '<span>保存密钥</span>';
        btn.style.opacity = '1';
    }
}

// 边框闪烁过渡效果
function flashElement(el, color) {
    const originalShadow = el.style.boxShadow;
    el.style.boxShadow = `0 0 16px ${color}`;
    setTimeout(() => {
        el.style.boxShadow = originalShadow;
    }, 800);
}

// 异步文本复制并渲染复制状态切换
function copyTextToClipboard(text, button) {
    if (!navigator.clipboard) {
        // Fallback for non-https/unsupported browsers
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        document.body.appendChild(textarea);
        textarea.focus();
        textarea.select();
        try {
            document.execCommand('copy');
            animateCopySuccess(button);
        } catch (err) {
            console.error('Fallback copy failed', err);
        }
        document.body.removeChild(textarea);
        return;
    }
    
    navigator.clipboard.writeText(text).then(() => {
        animateCopySuccess(button);
    }, (err) => {
        console.error('Could not copy text: ', err);
    });
}

function animateCopySuccess(button) {
    const span = button.querySelector('span');
    const originalText = span.textContent;
    
    span.textContent = '已复制！';
    button.style.background = 'rgba(16, 185, 129, 0.15)';
    button.style.borderColor = 'rgba(16, 185, 129, 0.4)';
    button.style.color = '#10b981';
    
    setTimeout(() => {
        span.textContent = originalText;
        button.style.background = '';
        button.style.borderColor = '';
        button.style.color = '';
    }, 1500);
}

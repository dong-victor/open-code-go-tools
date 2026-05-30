import os

path = 'frontend/app.js'
with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

# 1. Update i18n dictionary for zh and en
content = content.replace(
    'btn_edit_settings_json: "编辑 settings.json",',
    'btn_edit_settings_json: "编辑 settings.json",\n        btn_sync_models: "同步上游模型",'
)
content = content.replace(
    'btn_edit_settings_json: "Edit settings.json",',
    'btn_edit_settings_json: "Edit settings.json",\n        btn_sync_models: "Sync Models",'
)

# 2. Change const MODEL_REGISTRY to let MODEL_REGISTRY (if we want to replace it)
content = content.replace('const MODEL_REGISTRY = [', 'let MODEL_REGISTRY = [')

# 3. Add event listener for btn-sync-models
# Let's insert it inside the DOMContentLoaded block or setupEventListeners function.
# I will search for setupEventListeners() or similar initialization logic.
# A safe place to insert event listener is around other button listeners, e.g., document.getElementById('save-all-config-btn')

event_listener_code = """
    const syncModelsBtn = document.getElementById('btn-sync-models');
    if (syncModelsBtn) {
        syncModelsBtn.addEventListener('click', async () => {
            try {
                syncModelsBtn.disabled = true;
                const oldText = syncModelsBtn.textContent;
                syncModelsBtn.textContent = '...';
                const res = await fetch(`${API_BASE}/v1/models`);
                if (!res.ok) throw new Error('API failed');
                const data = await res.json();
                if (data && data.data && Array.isArray(data.data)) {
                    const newModels = data.data.map(m => ({
                        id: m.id,
                        label: m.id,
                        recommended: false,
                        category: 'Synced'
                    }));
                    // Keep original recommended models if not in the list, or just append new ones
                    const existingIds = new Set(MODEL_REGISTRY.map(m => m.id));
                    let added = 0;
                    for (const nm of newModels) {
                        if (!existingIds.has(nm.id)) {
                            MODEL_REGISTRY.push(nm);
                            added++;
                        }
                    }
                    populateModelSelects();
                    showToast(`同步成功，新增 ${added} 个模型`);
                }
            } catch (err) {
                console.error(err);
                showToast('获取模型失败，请检查上游 API Key 与网络连接', 'error');
            } finally {
                syncModelsBtn.disabled = false;
                syncModelsBtn.textContent = t('btn_sync_models');
            }
        });
    }
"""

if "document.getElementById('save-all-config-btn')" in content:
    content = content.replace("document.getElementById('save-all-config-btn')", event_listener_code + "\n    document.getElementById('save-all-config-btn')")

with open(path, 'w', encoding='utf-8') as f:
    f.write(content)


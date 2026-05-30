import os

def replace_in_file(path, replacements):
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    for old, new in replacements:
        content = content.replace(old, new)
    with open(path, 'w', encoding='utf-8') as f:
        f.write(content)

replace_in_file('RELEASE_NOTES.md', [
    ('重大更新版本发布', 'v2.0.0 版本说明 (Release Notes)'),
    ('本次 2.0.0 版本包含全面的系统升级，完整的 MD 文档更新与最新的控制面板截图。', '本次更新主要包括多平台构建支持、增加 Qwen3.7 模型、修复 Token 统计以及界面截图的同步更新。'),
    ('移除了旧版的 AI 生成界面截图，全面替换为最新的实机截图。', '更新了文档中的控制面板截图。'),
    ('更多底层稳定性改进。', '修复了若干代理服务的稳定性问题。'),
    ('Major Update Release', 'Release Notes for v2.0.0'),
    ('This 2.0.0 release includes comprehensive system upgrades, fully updated MD documentation, and the latest control panel screenshots.', 'This update introduces multi-platform builds, adds the Qwen3.7 model, fixes token usage stats, and updates interface screenshots.'),
    ('Removed old AI-generated screenshots, replaced with the latest actual screenshots.', 'Updated control panel screenshots in the documentation.'),
    ('More underlying stability improvements.', 'Fixed proxy service stability issues.'),
])

replace_in_file('README.md', [
    ('配合极简直觉的中英双语 GUI，一键拉起开发控制台。', '提供中英双语 GUI，支持一键启动配置终端。'),
    ('极简直觉', '简洁'),
    ('杜绝误配', '防止误配'),
    ('极简配置管理', '配置管理'),
    ('秒级热重载生效', '配置热重载生效'),
    ('一键终端唤醒', '终端启动'),
    ('流量雷达监控', '流量监控'),
])

replace_in_file('docs/README.en-US.md', [
    ('Premium Configuration Management', 'Configuration Management'),
    ('Traffic Radar Logs', 'Traffic Logs'),
])


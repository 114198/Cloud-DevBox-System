// ===== 系统设置页面 =====

function renderSettings() {
    const page = document.getElementById('settingsPage');
    const settings = USER_SETTINGS;
    
    page.innerHTML = `
        <div class="page-header">
            <div class="page-header-left">
                <h2>系统设置</h2>
                <p>管理您的账户和偏好设置</p>
            </div>
        </div>
        
        <div class="settings-layout">
            <!-- 设置导航 -->
            <div class="settings-nav">
                <button class="settings-nav-item active" onclick="switchSettingsTab('profile')">
                    <i class="fas fa-user"></i>
                    <span>个人资料</span>
                </button>
                <button class="settings-nav-item" onclick="switchSettingsTab('preferences')">
                    <i class="fas fa-sliders-h"></i>
                    <span>偏好设置</span>
                </button>
                <button class="settings-nav-item" onclick="switchSettingsTab('security')">
                    <i class="fas fa-shield-alt"></i>
                    <span>安全设置</span>
                </button>
                <button class="settings-nav-item" onclick="switchSettingsTab('ssh')">
                    <i class="fas fa-key"></i>
                    <span>SSH 密钥</span>
                </button>
                <button class="settings-nav-item" onclick="switchSettingsTab('api')">
                    <i class="fas fa-code"></i>
                    <span>API 密钥</span>
                </button>
                <button class="settings-nav-item" onclick="switchSettingsTab('notifications')">
                    <i class="fas fa-bell"></i>
                    <span>通知设置</span>
                </button>
            </div>
            
            <!-- 设置内容 -->
            <div class="settings-content">
                <!-- 个人资料 -->
                <div class="settings-panel active" id="profilePanel">
                    <h3>个人资料</h3>
                    <div class="profile-section">
                        <div class="avatar-upload">
                            <img src="${settings.profile.avatar}" alt="头像" class="avatar-preview">
                            <button class="btn btn-outline btn-sm">更换头像</button>
                        </div>
                        <div class="profile-form">
                            <div class="form-group">
                                <label>用户名</label>
                                <input type="text" class="form-input" value="${settings.profile.username}" readonly>
                            </div>
                            <div class="form-group">
                                <label>显示名称</label>
                                <input type="text" class="form-input" value="${settings.profile.displayName}">
                            </div>
                            <div class="form-group">
                                <label>邮箱</label>
                                <input type="email" class="form-input" value="${settings.profile.email}">
                            </div>
                            <div class="form-group">
                                <label>组织</label>
                                <input type="text" class="form-input" value="${settings.profile.organization || ''}" placeholder="输入组织名称">
                            </div>
                            <button class="btn btn-primary" onclick="saveProfile()">保存更改</button>
                        </div>
                    </div>
                </div>
                
                <!-- 偏好设置 -->
                <div class="settings-panel" id="preferencesPanel">
                    <h3>偏好设置</h3>
                    <div class="form-group">
                        <label>界面语言</label>
                        <select class="form-input" id="languageSetting">
                            <option value="zh-CN" ${settings.preferences.language === 'zh-CN' ? 'selected' : ''}>简体中文</option>
                            <option value="zh-TW" ${settings.preferences.language === 'zh-TW' ? 'selected' : ''}>繁體中文</option>
                            <option value="en" ${settings.preferences.language === 'en' ? 'selected' : ''}>English</option>
                            <option value="ja" ${settings.preferences.language === 'ja' ? 'selected' : ''}>日本語</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>时区</label>
                        <select class="form-input" id="timezoneSetting">
                            <option value="Asia/Shanghai" ${settings.preferences.timezone === 'Asia/Shanghai' ? 'selected' : ''}>中国标准时间 (UTC+8)</option>
                            <option value="Asia/Tokyo" ${settings.preferences.timezone === 'Asia/Tokyo' ? 'selected' : ''}>日本标准时间 (UTC+9)</option>
                            <option value="America/New_York" ${settings.preferences.timezone === 'America/New_York' ? 'selected' : ''}>美国东部时间 (UTC-5)</option>
                            <option value="Europe/London" ${settings.preferences.timezone === 'Europe/London' ? 'selected' : ''}>格林威治时间 (UTC+0)</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>主题</label>
                        <div class="theme-options">
                            <label class="theme-option ${settings.preferences.theme === 'dark' ? 'selected' : ''}">
                                <input type="radio" name="theme" value="dark" ${settings.preferences.theme === 'dark' ? 'checked' : ''}>
                                <div class="theme-preview dark">
                                    <i class="fas fa-moon"></i>
                                </div>
                                <span>深色</span>
                            </label>
                            <label class="theme-option ${settings.preferences.theme === 'light' ? 'selected' : ''}">
                                <input type="radio" name="theme" value="light" ${settings.preferences.theme === 'light' ? 'checked' : ''}>
                                <div class="theme-preview light">
                                    <i class="fas fa-sun"></i>
                                </div>
                                <span>浅色</span>
                            </label>
                            <label class="theme-option">
                                <input type="radio" name="theme" value="auto">
                                <div class="theme-preview auto">
                                    <i class="fas fa-adjust"></i>
                                </div>
                                <span>跟随系统</span>
                            </label>
                        </div>
                    </div>
                    <button class="btn btn-primary" onclick="savePreferences()">保存设置</button>
                </div>
                
                <!-- 安全设置 -->
                <div class="settings-panel" id="securityPanel">
                    <h3>安全设置</h3>
                    <div class="security-section">
                        <div class="security-item">
                            <div class="security-info">
                                <h4>修改密码</h4>
                                <p>上次修改: ${settings.security.lastPasswordChange}</p>
                            </div>
                            <button class="btn btn-outline" onclick="openChangePasswordModal()">修改密码</button>
                        </div>
                        <div class="security-item">
                            <div class="security-info">
                                <h4>两步验证</h4>
                                <p>${settings.security.mfaEnabled ? '已启用' : '未启用'}</p>
                            </div>
                            <button class="btn ${settings.security.mfaEnabled ? 'btn-danger' : 'btn-primary'}" onclick="toggleMFA()">
                                ${settings.security.mfaEnabled ? '禁用' : '启用'}
                            </button>
                        </div>
                        <div class="security-item">
                            <div class="security-info">
                                <h4>活跃会话</h4>
                                <p>当前有 ${settings.security.activeSessions} 个活跃会话</p>
                            </div>
                            <button class="btn btn-outline" onclick="viewSessions()">管理会话</button>
                        </div>
                    </div>
                </div>
                
                <!-- SSH 密钥 -->
                <div class="settings-panel" id="sshPanel">
                    <h3>SSH 密钥</h3>
                    <p class="panel-desc">管理用于连接开发环境的 SSH 密钥</p>
                    <div class="keys-list">
                        ${settings.sshKeys.map(key => `
                            <div class="key-item">
                                <div class="key-icon">
                                    <i class="fas fa-key"></i>
                                </div>
                                <div class="key-info">
                                    <span class="key-name">${key.name}</span>
                                    <span class="key-fingerprint">${key.fingerprint}</span>
                                    <span class="key-date">添加于 ${key.createdAt}</span>
                                </div>
                                <button class="btn-icon" onclick="deleteSSHKey('${key.id}')" title="删除">
                                    <i class="fas fa-trash"></i>
                                </button>
                            </div>
                        `).join('')}
                    </div>
                    <button class="btn btn-primary" onclick="openAddSSHKeyModal()">
                        <i class="fas fa-plus"></i> 添加 SSH 密钥
                    </button>
                </div>
                
                <!-- API 密钥 -->
                <div class="settings-panel" id="apiPanel">
                    <h3>API 密钥</h3>
                    <p class="panel-desc">管理用于 API 访问的密钥</p>
                    <div class="keys-list">
                        ${settings.apiKeys.map(key => `
                            <div class="key-item">
                                <div class="key-icon">
                                    <i class="fas fa-code"></i>
                                </div>
                                <div class="key-info">
                                    <span class="key-name">${key.name}</span>
                                    <span class="key-fingerprint">${key.prefix}****</span>
                                    <span class="key-date">创建于 ${key.createdAt} · 最后使用 ${key.lastUsed}</span>
                                </div>
                                <button class="btn-icon" onclick="deleteAPIKey('${key.id}')" title="删除">
                                    <i class="fas fa-trash"></i>
                                </button>
                            </div>
                        `).join('')}
                    </div>
                    <button class="btn btn-primary" onclick="openCreateAPIKeyModal()">
                        <i class="fas fa-plus"></i> 创建 API 密钥
                    </button>
                </div>
                
                <!-- 通知设置 -->
                <div class="settings-panel" id="notificationsPanel">
                    <h3>通知设置</h3>
                    <div class="notification-settings">
                        <div class="notification-item">
                            <div class="notification-info">
                                <h4>邮件通知</h4>
                                <p>接收重要通知的邮件提醒</p>
                            </div>
                            <label class="toggle-switch">
                                <input type="checkbox" ${settings.preferences.notifications.email ? 'checked' : ''}>
                                <span class="toggle-slider"></span>
                            </label>
                        </div>
                        <div class="notification-item">
                            <div class="notification-info">
                                <h4>浏览器通知</h4>
                                <p>在浏览器中显示桌面通知</p>
                            </div>
                            <label class="toggle-switch">
                                <input type="checkbox" ${settings.preferences.notifications.browser ? 'checked' : ''}>
                                <span class="toggle-slider"></span>
                            </label>
                        </div>
                        <div class="notification-item">
                            <div class="notification-info">
                                <h4>部署成功通知</h4>
                                <p>部署完成时发送通知</p>
                            </div>
                            <label class="toggle-switch">
                                <input type="checkbox" ${settings.preferences.notifications.deploySuccess ? 'checked' : ''}>
                                <span class="toggle-slider"></span>
                            </label>
                        </div>
                        <div class="notification-item">
                            <div class="notification-info">
                                <h4>部署失败通知</h4>
                                <p>部署失败时发送通知</p>
                            </div>
                            <label class="toggle-switch">
                                <input type="checkbox" ${settings.preferences.notifications.deployFailed ? 'checked' : ''}>
                                <span class="toggle-slider"></span>
                            </label>
                        </div>
                        <div class="notification-item">
                            <div class="notification-info">
                                <h4>资源告警</h4>
                                <p>资源使用超限时发送告警</p>
                            </div>
                            <label class="toggle-switch">
                                <input type="checkbox" ${settings.preferences.notifications.resourceAlert ? 'checked' : ''}>
                                <span class="toggle-slider"></span>
                            </label>
                        </div>
                    </div>
                    <button class="btn btn-primary" onclick="saveNotificationSettings()">保存设置</button>
                </div>
            </div>
        </div>
    `;
    
    addSettingsStyles();
}

function switchSettingsTab(tabName) {
    document.querySelectorAll('.settings-nav-item').forEach(item => {
        item.classList.remove('active');
    });
    document.querySelectorAll('.settings-panel').forEach(panel => {
        panel.classList.remove('active');
    });
    
    event.currentTarget.classList.add('active');
    document.getElementById(`${tabName}Panel`).classList.add('active');
}

function saveProfile() {
    showToast('个人资料已保存', 'success');
}

function savePreferences() {
    showToast('偏好设置已保存', 'success');
}

function openChangePasswordModal() {
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'changePasswordModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('changePasswordModal')"></div>
        <div class="modal-content">
            <div class="modal-header">
                <h3>修改密码</h3>
                <button class="modal-close" onclick="closeModal('changePasswordModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="form-group">
                    <label>当前密码</label>
                    <input type="password" class="form-input" placeholder="输入当前密码">
                </div>
                <div class="form-group">
                    <label>新密码</label>
                    <input type="password" class="form-input" placeholder="输入新密码">
                </div>
                <div class="form-group">
                    <label>确认新密码</label>
                    <input type="password" class="form-input" placeholder="再次输入新密码">
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-outline" onclick="closeModal('changePasswordModal')">取消</button>
                <button class="btn btn-primary" onclick="changePassword()">确认修改</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function changePassword() {
    closeModal('changePasswordModal');
    showToast('密码已修改', 'success');
}

function toggleMFA() {
    USER_SETTINGS.security.mfaEnabled = !USER_SETTINGS.security.mfaEnabled;
    showToast(USER_SETTINGS.security.mfaEnabled ? '两步验证已启用' : '两步验证已禁用', 'success');
    renderSettings();
}

function viewSessions() {
    showToast('查看活跃会话', 'info');
}

function openAddSSHKeyModal() {
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'addSSHKeyModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('addSSHKeyModal')"></div>
        <div class="modal-content">
            <div class="modal-header">
                <h3>添加 SSH 密钥</h3>
                <button class="modal-close" onclick="closeModal('addSSHKeyModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="form-group">
                    <label>密钥名称</label>
                    <input type="text" class="form-input" placeholder="例如: MacBook Pro">
                </div>
                <div class="form-group">
                    <label>公钥内容</label>
                    <textarea class="form-input" rows="5" placeholder="粘贴您的 SSH 公钥 (ssh-rsa 或 ssh-ed25519)"></textarea>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-outline" onclick="closeModal('addSSHKeyModal')">取消</button>
                <button class="btn btn-primary" onclick="addSSHKey()">添加密钥</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function addSSHKey() {
    closeModal('addSSHKeyModal');
    showToast('SSH 密钥已添加', 'success');
}

function deleteSSHKey(keyId) {
    if (confirm('确定要删除这个 SSH 密钥吗？')) {
        showToast('SSH 密钥已删除', 'success');
    }
}

function openCreateAPIKeyModal() {
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'createAPIKeyModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('createAPIKeyModal')"></div>
        <div class="modal-content">
            <div class="modal-header">
                <h3>创建 API 密钥</h3>
                <button class="modal-close" onclick="closeModal('createAPIKeyModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="form-group">
                    <label>密钥名称</label>
                    <input type="text" class="form-input" placeholder="例如: CI/CD 集成">
                </div>
                <div class="form-group">
                    <label>权限范围</label>
                    <div class="permission-checkboxes">
                        <label class="checkbox-label">
                            <input type="checkbox" checked> 读取环境信息
                        </label>
                        <label class="checkbox-label">
                            <input type="checkbox" checked> 管理环境
                        </label>
                        <label class="checkbox-label">
                            <input type="checkbox"> 部署应用
                        </label>
                    </div>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-outline" onclick="closeModal('createAPIKeyModal')">取消</button>
                <button class="btn btn-primary" onclick="createAPIKey()">创建密钥</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function createAPIKey() {
    closeModal('createAPIKeyModal');
    showToast('API 密钥已创建', 'success');
}

function deleteAPIKey(keyId) {
    if (confirm('确定要删除这个 API 密钥吗？')) {
        showToast('API 密钥已删除', 'success');
    }
}

function saveNotificationSettings() {
    showToast('通知设置已保存', 'success');
}

function addSettingsStyles() {
    if (document.getElementById('settingsStyles')) return;
    
    const style = document.createElement('style');
    style.id = 'settingsStyles';
    style.textContent = `
        .settings-layout {
            display: grid;
            grid-template-columns: 250px 1fr;
            gap: 24px;
        }
        
        .settings-nav {
            display: flex;
            flex-direction: column;
            gap: 4px;
            background: var(--bg-base);
            border: 1px solid var(--border-default);
            border-radius: var(--radius-lg);
            padding: 12px;
            height: fit-content;
        }
        
        .settings-nav-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 16px;
            border-radius: var(--radius-md);
            color: var(--text-secondary);
            text-align: left;
            transition: all var(--transition-fast);
        }
        
        .settings-nav-item:hover {
            background: var(--bg-elevated);
            color: var(--text-primary);
        }
        
        .settings-nav-item.active {
            background: linear-gradient(135deg, var(--primary), var(--primary-dark));
            color: white;
        }
        
        .settings-content {
            background: var(--bg-base);
            border: 1px solid var(--border-default);
            border-radius: var(--radius-lg);
            padding: 24px;
        }
        
        .settings-panel {
            display: none;
        }
        
        .settings-panel.active {
            display: block;
        }
        
        .settings-panel h3 {
            margin-bottom: 24px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--border-default);
        }
        
        .panel-desc {
            color: var(--text-secondary);
            margin-bottom: 20px;
        }
        
        .profile-section {
            display: flex;
            gap: 40px;
        }
        
        .avatar-upload {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 12px;
        }
        
        .avatar-preview {
            width: 120px;
            height: 120px;
            border-radius: 50%;
            border: 4px solid var(--border-default);
        }
        
        .profile-form {
            flex: 1;
            max-width: 400px;
        }
        
        .theme-options {
            display: flex;
            gap: 16px;
        }
        
        .theme-option {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 8px;
            cursor: pointer;
        }
        
        .theme-option input {
            display: none;
        }
        
        .theme-preview {
            width: 80px;
            height: 60px;
            display: flex;
            align-items: center;
            justify-content: center;
            border: 2px solid var(--border-default);
            border-radius: var(--radius-md);
            font-size: 24px;
            transition: all var(--transition-fast);
        }
        
        .theme-preview.dark {
            background: #161B22;
            color: #E6EDF3;
        }
        
        .theme-preview.light {
            background: #f8fafc;
            color: #1e293b;
        }
        
        .theme-preview.auto {
            background: linear-gradient(135deg, #161B22 50%, #f8fafc 50%);
            color: var(--primary);
        }
        
        .theme-option.selected .theme-preview {
            border-color: var(--primary);
        }
        
        .security-section {
            display: flex;
            flex-direction: column;
            gap: 16px;
        }
        
        .security-item {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 16px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .security-info h4 {
            margin-bottom: 4px;
        }
        
        .security-info p {
            font-size: 13px;
            color: var(--text-tertiary);
        }
        
        .keys-list {
            display: flex;
            flex-direction: column;
            gap: 12px;
            margin-bottom: 20px;
        }
        
        .key-item {
            display: flex;
            align-items: center;
            gap: 16px;
            padding: 16px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .key-icon {
            width: 40px;
            height: 40px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: rgba(54,173,239,0.15);
            border-radius: var(--radius-md);
            color: var(--primary);
        }
        
        .key-info {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        
        .key-name {
            font-weight: 500;
        }
        
        .key-fingerprint {
            font-family: 'SF Mono', 'Fira Code', monospace;
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .key-date {
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .notification-settings {
            display: flex;
            flex-direction: column;
            gap: 16px;
            margin-bottom: 20px;
        }
        
        .notification-item {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 16px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .notification-info h4 {
            margin-bottom: 4px;
        }
        
        .notification-info p {
            font-size: 13px;
            color: var(--text-tertiary);
        }
        
        .toggle-switch {
            position: relative;
            width: 48px;
            height: 24px;
        }
        
        .toggle-switch input {
            opacity: 0;
            width: 0;
            height: 0;
        }
        
        .toggle-slider {
            position: absolute;
            cursor: pointer;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background-color: var(--bg-surface);
            border-radius: 24px;
            transition: all var(--transition-fast);
        }
        
        .toggle-slider:before {
            position: absolute;
            content: "";
            height: 18px;
            width: 18px;
            left: 3px;
            bottom: 3px;
            background-color: white;
            border-radius: 50%;
            transition: all var(--transition-fast);
        }
        
        .toggle-switch input:checked + .toggle-slider {
            background-color: var(--primary);
        }
        
        .toggle-switch input:checked + .toggle-slider:before {
            transform: translateX(24px);
        }
        
        .permission-checkboxes {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }
        
        @media (max-width: 768px) {
            .settings-layout {
                grid-template-columns: 1fr;
            }
            
            .settings-nav {
                flex-direction: row;
                overflow-x: auto;
            }
            
            .settings-nav-item span {
                display: none;
            }
            
            .profile-section {
                flex-direction: column;
            }
        }
    `;
    document.head.appendChild(style);
}

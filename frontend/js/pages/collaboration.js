// ===== 协作开发页面 =====

function renderCollaboration() {
    const page = document.getElementById('collaborationPage');
    
    page.innerHTML = `
        <div class="page-header">
            <div class="page-header-left">
                <h2>协作开发</h2>
                <p>实时协作编辑和团队协作管理</p>
            </div>
        </div>
        
        <!-- 活跃会话 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">
                    <i class="fas fa-circle text-success pulse"></i>
                    活跃协作会话
                </h3>
            </div>
            <div class="card-body">
                ${COLLABORATION_SESSIONS.length > 0 ? `
                    <div class="collab-sessions">
                        ${COLLABORATION_SESSIONS.map(session => `
                            <div class="collab-session-card">
                                <div class="session-header">
                                    <div class="session-env">
                                        <i class="fas fa-server"></i>
                                        <span>${session.envName}</span>
                                    </div>
                                    <span class="session-time">
                                        <i class="fas fa-clock"></i>
                                        开始于 ${formatRelativeTime(session.startTime)}
                                    </span>
                                </div>
                                
                                <div class="session-participants">
                                    <h4>参与者 (${session.participants.length})</h4>
                                    <div class="participants-list">
                                        ${session.participants.map(p => `
                                            <div class="participant ${p.status}">
                                                <img src="https://api.dicebear.com/7.x/avataaars/svg?seed=${p.name}" alt="${p.name}" class="participant-avatar">
                                                <div class="participant-info">
                                                    <span class="participant-name">${p.name}</span>
                                                    <span class="participant-location">
                                                        <i class="fas fa-file-code"></i>
                                                        ${p.cursor.file}:${p.cursor.line}
                                                    </span>
                                                </div>
                                                <span class="participant-status ${p.status}">
                                                    ${p.status === 'active' ? '编辑中' : '空闲'}
                                                </span>
                                            </div>
                                        `).join('')}
                                    </div>
                                </div>
                                
                                <div class="session-files">
                                    <h4>活跃文件</h4>
                                    <div class="files-list">
                                        ${session.activeFiles.map(file => `
                                            <div class="file-item">
                                                <i class="fas fa-file-code"></i>
                                                <span>${file}</span>
                                            </div>
                                        `).join('')}
                                    </div>
                                </div>
                                
                                <div class="session-actions">
                                    <button class="btn btn-primary" onclick="joinSession('${session.id}')">
                                        <i class="fas fa-sign-in-alt"></i> 加入会话
                                    </button>
                                    <button class="btn btn-outline" onclick="viewSessionHistory('${session.id}')">
                                        <i class="fas fa-history"></i> 查看历史
                                    </button>
                                </div>
                            </div>
                        `).join('')}
                    </div>
                ` : `
                    <div class="empty-state">
                        <i class="fas fa-users"></i>
                        <h3>暂无活跃会话</h3>
                        <p>开始一个新的协作会话</p>
                    </div>
                `}
            </div>
        </div>
        
        <!-- 邀请协作 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">邀请协作</h3>
            </div>
            <div class="card-body">
                <div class="invite-section">
                    <div class="form-group">
                        <label>选择环境</label>
                        <select id="collabEnvSelect" class="form-input">
                            ${ENVIRONMENTS.filter(e => e.status === 'running').map(env => `
                                <option value="${env.id}">${env.name}</option>
                            `).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label>邀请方式</label>
                        <div class="invite-methods">
                            <div class="invite-method" onclick="generateInviteLink()">
                                <i class="fas fa-link"></i>
                                <span>生成邀请链接</span>
                            </div>
                            <div class="invite-method" onclick="inviteByEmail()">
                                <i class="fas fa-envelope"></i>
                                <span>邮件邀请</span>
                            </div>
                        </div>
                    </div>
                    <div class="invite-link-box" id="inviteLinkBox" style="display: none;">
                        <input type="text" id="inviteLink" class="form-input" readonly>
                        <button class="btn btn-primary" onclick="copyInviteLink()">
                            <i class="fas fa-copy"></i> 复制
                        </button>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 协作权限 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">协作权限管理</h3>
            </div>
            <div class="card-body">
                <div class="permissions-info">
                    <div class="permission-item">
                        <div class="permission-icon owner">
                            <i class="fas fa-crown"></i>
                        </div>
                        <div class="permission-content">
                            <h4>所有者</h4>
                            <p>完全控制权限，可以管理成员、修改设置、删除环境</p>
                        </div>
                    </div>
                    <div class="permission-item">
                        <div class="permission-icon editor">
                            <i class="fas fa-edit"></i>
                        </div>
                        <div class="permission-content">
                            <h4>编辑者</h4>
                            <p>可以编辑代码、运行命令、部署应用</p>
                        </div>
                    </div>
                    <div class="permission-item">
                        <div class="permission-icon viewer">
                            <i class="fas fa-eye"></i>
                        </div>
                        <div class="permission-content">
                            <h4>查看者</h4>
                            <p>只读权限，可以查看代码和日志</p>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    addCollaborationStyles();
}

function joinSession(sessionId) {
    showToast('正在加入协作会话...', 'info');
    setTimeout(() => {
        showToast('已加入协作会话', 'success');
    }, 1000);
}

function viewSessionHistory(sessionId) {
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'sessionHistoryModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('sessionHistoryModal')"></div>
        <div class="modal-content modal-lg">
            <div class="modal-header">
                <h3>协作历史</h3>
                <button class="modal-close" onclick="closeModal('sessionHistoryModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="history-timeline">
                    <div class="history-item">
                        <div class="history-time">10:30</div>
                        <div class="history-content">
                            <span class="history-user">张三</span>
                            <span class="history-action">修改了</span>
                            <span class="history-file">src/App.tsx</span>
                        </div>
                    </div>
                    <div class="history-item">
                        <div class="history-time">10:28</div>
                        <div class="history-content">
                            <span class="history-user">李四</span>
                            <span class="history-action">创建了</span>
                            <span class="history-file">src/components/Header.tsx</span>
                        </div>
                    </div>
                    <div class="history-item">
                        <div class="history-time">10:25</div>
                        <div class="history-content">
                            <span class="history-user">张三</span>
                            <span class="history-action">加入了会话</span>
                        </div>
                    </div>
                    <div class="history-item">
                        <div class="history-time">10:20</div>
                        <div class="history-content">
                            <span class="history-user">李四</span>
                            <span class="history-action">开始了协作会话</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function generateInviteLink() {
    const envId = document.getElementById('collabEnvSelect').value;
    const link = `https://devbox.io/invite/${envId}/${Math.random().toString(36).substr(2, 8)}`;
    
    document.getElementById('inviteLink').value = link;
    document.getElementById('inviteLinkBox').style.display = 'flex';
    
    showToast('邀请链接已生成', 'success');
}

function copyInviteLink() {
    const link = document.getElementById('inviteLink').value;
    copyToClipboard(link);
}

function inviteByEmail() {
    const email = prompt('请输入要邀请的邮箱地址：');
    if (email) {
        showToast(`邀请已发送至 ${email}`, 'success');
    }
}

function addCollaborationStyles() {
    if (document.getElementById('collaborationStyles')) return;
    
    const style = document.createElement('style');
    style.id = 'collaborationStyles';
    style.textContent = `
        .text-success {
            color: var(--success);
        }
        
        .pulse {
            animation: pulse 2s infinite;
        }
        
        .collab-sessions {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
            gap: 20px;
        }
        
        .collab-session-card {
            background: var(--bg-elevated);
            border-radius: var(--radius-lg);
            padding: 20px;
        }
        
        .session-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--border-default);
        }
        
        .session-env {
            display: flex;
            align-items: center;
            gap: 8px;
            font-weight: 600;
            font-size: 16px;
        }
        
        .session-time {
            font-size: 13px;
            color: var(--text-tertiary);
        }
        
        .session-participants h4,
        .session-files h4 {
            font-size: 13px;
            color: var(--text-secondary);
            margin-bottom: 12px;
        }
        
        .participants-list {
            display: flex;
            flex-direction: column;
            gap: 12px;
            margin-bottom: 20px;
        }
        
        .participant {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 10px;
            background: var(--bg-base);
            border-radius: var(--radius-md);
        }
        
        .participant-avatar {
            width: 36px;
            height: 36px;
            border-radius: 50%;
        }
        
        .participant-info {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        
        .participant-name {
            font-weight: 500;
        }
        
        .participant-location {
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .participant-status {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 11px;
        }
        
        .participant-status.active {
            background: rgba(0,217,181,0.15);
            color: var(--success);
        }
        
        .participant-status.idle {
            background: rgba(139,148,158,0.15);
            color: var(--text-tertiary);
        }
        
        .files-list {
            display: flex;
            flex-wrap: wrap;
            gap: 8px;
            margin-bottom: 20px;
        }
        
        .file-item {
            display: flex;
            align-items: center;
            gap: 6px;
            padding: 6px 10px;
            background: var(--bg-base);
            border-radius: 4px;
            font-size: 12px;
            color: var(--text-secondary);
        }
        
        .session-actions {
            display: flex;
            gap: 12px;
        }
        
        .invite-section {
            max-width: 500px;
        }
        
        .invite-methods {
            display: flex;
            gap: 12px;
        }
        
        .invite-method {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 8px;
            padding: 20px;
            background: var(--bg-elevated);
            border: 1px solid var(--border-default);
            border-radius: var(--radius-md);
            cursor: pointer;
            transition: all var(--transition-fast);
        }
        
        .invite-method:hover {
            border-color: var(--primary);
        }
        
        .invite-method i {
            font-size: 24px;
            color: var(--primary);
        }
        
        .invite-link-box {
            display: flex;
            gap: 12px;
            margin-top: 16px;
        }
        
        .invite-link-box .form-input {
            flex: 1;
        }
        
        .permissions-info {
            display: flex;
            flex-direction: column;
            gap: 16px;
        }
        
        .permission-item {
            display: flex;
            gap: 16px;
            padding: 16px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .permission-icon {
            width: 48px;
            height: 48px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: var(--radius-md);
            font-size: 20px;
        }
        
        .permission-icon.owner {
            background: rgba(255,176,32,0.15);
            color: var(--warning);
        }
        
        .permission-icon.editor {
            background: rgba(54,173,239,0.15);
            color: var(--primary);
        }
        
        .permission-icon.viewer {
            background: rgba(139,148,158,0.15);
            color: var(--text-tertiary);
        }
        
        .permission-content h4 {
            margin-bottom: 4px;
        }
        
        .permission-content p {
            font-size: 13px;
            color: var(--text-secondary);
        }
        
        .history-timeline {
            display: flex;
            flex-direction: column;
            gap: 16px;
        }
        
        .history-item {
            display: flex;
            gap: 16px;
            padding: 12px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .history-time {
            font-size: 13px;
            color: var(--text-tertiary);
            min-width: 50px;
        }
        
        .history-content {
            font-size: 14px;
        }
        
        .history-user {
            font-weight: 500;
            color: var(--primary);
        }
        
        .history-action {
            color: var(--text-secondary);
        }
        
        .history-file {
            font-family: 'SF Mono', 'Fira Code', monospace;
            background: var(--bg-base);
            padding: 2px 6px;
            border-radius: 4px;
        }
    `;
    document.head.appendChild(style);
}

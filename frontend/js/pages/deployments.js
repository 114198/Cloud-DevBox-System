// ===== 部署管理页面 =====

function renderDeployments() {
    const page = document.getElementById('deploymentsPage');
    
    page.innerHTML = `
        <div class="page-header">
            <div class="page-header-left">
                <h2>部署管理</h2>
                <p>查看和管理应用部署记录</p>
            </div>
            <div class="page-header-right">
                <button class="btn btn-primary" onclick="openDeployModal()">
                    <i class="fas fa-rocket"></i> 新建部署
                </button>
            </div>
        </div>
        
        <!-- 部署统计 -->
        <div class="deploy-stats">
            <div class="deploy-stat-card">
                <div class="stat-icon green">
                    <i class="fas fa-check-circle"></i>
                </div>
                <div class="stat-content">
                    <span class="stat-value">${DEPLOYMENTS.filter(d => d.status === 'success').length}</span>
                    <span class="stat-label">成功部署</span>
                </div>
            </div>
            <div class="deploy-stat-card">
                <div class="stat-icon red">
                    <i class="fas fa-times-circle"></i>
                </div>
                <div class="stat-content">
                    <span class="stat-value">${DEPLOYMENTS.filter(d => d.status === 'failed').length}</span>
                    <span class="stat-label">失败部署</span>
                </div>
            </div>
            <div class="deploy-stat-card">
                <div class="stat-icon blue">
                    <i class="fas fa-clock"></i>
                </div>
                <div class="stat-content">
                    <span class="stat-value">3m 45s</span>
                    <span class="stat-label">平均部署时间</span>
                </div>
            </div>
            <div class="deploy-stat-card">
                <div class="stat-icon purple">
                    <i class="fas fa-percentage"></i>
                </div>
                <div class="stat-content">
                    <span class="stat-value">${Math.round(DEPLOYMENTS.filter(d => d.status === 'success').length / DEPLOYMENTS.length * 100)}%</span>
                    <span class="stat-label">成功率</span>
                </div>
            </div>
        </div>
        
        <!-- 部署列表 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">部署历史</h3>
                <div class="card-actions">
                    <select class="form-input" style="width: 150px;" onchange="filterDeployments(this.value)">
                        <option value="all">全部状态</option>
                        <option value="success">成功</option>
                        <option value="failed">失败</option>
                    </select>
                </div>
            </div>
            <div class="card-body">
                <div class="table-container">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>环境</th>
                                <th>版本</th>
                                <th>状态</th>
                                <th>提交信息</th>
                                <th>触发者</th>
                                <th>耗时</th>
                                <th>时间</th>
                                <th>操作</th>
                            </tr>
                        </thead>
                        <tbody id="deploymentsTableBody">
                            ${renderDeploymentRows(DEPLOYMENTS)}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    `;
    
    addDeploymentsStyles();
}

function renderDeploymentRows(deployments) {
    return deployments.map(deploy => `
        <tr>
            <td>
                <div class="deploy-env">
                    <span class="env-name">${deploy.envName}</span>
                </div>
            </td>
            <td>
                <span class="version-tag">${deploy.version}</span>
            </td>
            <td>
                <span class="status-badge ${deploy.status}">
                    <i class="fas ${deploy.status === 'success' ? 'fa-check-circle' : 'fa-times-circle'}"></i>
                    ${deploy.status === 'success' ? '成功' : '失败'}
                </span>
            </td>
            <td>
                <div class="commit-info">
                    <code class="commit-id">${deploy.commitId}</code>
                    <span class="commit-msg">${deploy.commitMessage}</span>
                </div>
            </td>
            <td>${deploy.triggeredBy}</td>
            <td>${deploy.duration}</td>
            <td>${formatRelativeTime(deploy.startTime)}</td>
            <td>
                <div class="table-actions">
                    <button class="btn-icon" onclick="viewDeployLog('${deploy.id}')" title="查看日志">
                        <i class="fas fa-file-alt"></i>
                    </button>
                    ${deploy.status === 'success' ? `
                        <button class="btn-icon" onclick="rollbackDeploy('${deploy.id}')" title="回滚">
                            <i class="fas fa-undo"></i>
                        </button>
                    ` : `
                        <button class="btn-icon" onclick="retryDeploy('${deploy.id}')" title="重试">
                            <i class="fas fa-redo"></i>
                        </button>
                    `}
                </div>
            </td>
        </tr>
    `).join('');
}

function filterDeployments(status) {
    const filtered = status === 'all' 
        ? DEPLOYMENTS 
        : DEPLOYMENTS.filter(d => d.status === status);
    
    document.getElementById('deploymentsTableBody').innerHTML = renderDeploymentRows(filtered);
}

function openDeployModal() {
    const runningEnvs = ENVIRONMENTS.filter(e => e.status === 'running');
    
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'deployModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('deployModal')"></div>
        <div class="modal-content">
            <div class="modal-header">
                <h3>新建部署</h3>
                <button class="modal-close" onclick="closeModal('deployModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="form-group">
                    <label>选择环境</label>
                    <select id="deployEnvSelect" class="form-input">
                        ${runningEnvs.map(env => `
                            <option value="${env.id}">${env.name} (${env.templateName})</option>
                        `).join('')}
                    </select>
                </div>
                <div class="form-group">
                    <label>版本标签</label>
                    <input type="text" id="deployVersion" class="form-input" placeholder="v1.0.0">
                </div>
                <div class="form-group">
                    <label>部署配置</label>
                    <div class="deploy-options">
                        <label class="checkbox-label">
                            <input type="checkbox" checked> 自动构建 Docker 镜像
                        </label>
                        <label class="checkbox-label">
                            <input type="checkbox" checked> 运行测试
                        </label>
                        <label class="checkbox-label">
                            <input type="checkbox"> 启用蓝绿部署
                        </label>
                    </div>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-outline" onclick="closeModal('deployModal')">取消</button>
                <button class="btn btn-primary" onclick="startDeploy()">
                    <i class="fas fa-rocket"></i> 开始部署
                </button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function startDeploy() {
    const envId = document.getElementById('deployEnvSelect').value;
    const version = document.getElementById('deployVersion').value || 'v' + Date.now();
    const env = ENVIRONMENTS.find(e => e.id === envId);
    
    closeModal('deployModal');
    showToast('部署已开始...', 'info');
    
    // 模拟部署过程
    setTimeout(() => {
        const newDeploy = {
            id: 'deploy-' + Date.now(),
            envId: envId,
            envName: env.name,
            version: version,
            status: 'success',
            startTime: new Date().toISOString(),
            endTime: new Date().toISOString(),
            duration: '2分30秒',
            triggeredBy: USER_SETTINGS.profile.displayName,
            commitId: Math.random().toString(36).substr(2, 7),
            commitMessage: '手动部署'
        };
        
        DEPLOYMENTS.unshift(newDeploy);
        showToast('部署成功！', 'success');
        renderDeployments();
    }, 3000);
}

function viewDeployLog(deployId) {
    const deploy = DEPLOYMENTS.find(d => d.id === deployId);
    if (!deploy) return;
    
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'deployLogModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('deployLogModal')"></div>
        <div class="modal-content modal-lg">
            <div class="modal-header">
                <h3>部署日志 - ${deploy.envName} ${deploy.version}</h3>
                <button class="modal-close" onclick="closeModal('deployLogModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="deploy-log">
                    <pre class="log-content">[${deploy.startTime}] 开始部署...
[${deploy.startTime}] 拉取最新代码...
[${deploy.startTime}] 检出提交: ${deploy.commitId}
[${deploy.startTime}] 安装依赖...
[${deploy.startTime}] npm install completed
[${deploy.startTime}] 运行测试...
[${deploy.startTime}] All tests passed
[${deploy.startTime}] 构建 Docker 镜像...
[${deploy.startTime}] Building image: devbox/${deploy.envName}:${deploy.version}
[${deploy.startTime}] 推送镜像到仓库...
[${deploy.startTime}] 更新 Kubernetes 部署...
[${deploy.startTime}] Deployment updated successfully
[${deploy.endTime}] ${deploy.status === 'success' ? '✓ 部署完成' : '✗ 部署失败: ' + (deploy.errorMessage || '未知错误')}</pre>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-outline" onclick="closeModal('deployLogModal')">关闭</button>
                <button class="btn btn-primary" onclick="downloadLog('${deployId}')">
                    <i class="fas fa-download"></i> 下载日志
                </button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function rollbackDeploy(deployId) {
    if (confirm('确定要回滚到此版本吗？')) {
        showToast('正在回滚...', 'info');
        setTimeout(() => {
            showToast('回滚成功！', 'success');
        }, 2000);
    }
}

function retryDeploy(deployId) {
    showToast('正在重试部署...', 'info');
    setTimeout(() => {
        showToast('部署成功！', 'success');
        const deploy = DEPLOYMENTS.find(d => d.id === deployId);
        if (deploy) {
            deploy.status = 'success';
            renderDeployments();
        }
    }, 3000);
}

function downloadLog(deployId) {
    showToast('正在下载日志...', 'info');
}

function addDeploymentsStyles() {
    if (document.getElementById('deploymentsStyles')) return;
    
    const style = document.createElement('style');
    style.id = 'deploymentsStyles';
    style.textContent = `
        .deploy-stats {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 20px;
            margin-bottom: 24px;
        }
        
        .deploy-stat-card {
            display: flex;
            align-items: center;
            gap: 16px;
            padding: 20px;
            background: var(--bg-base);
            border: 1px solid var(--border-default);
            border-radius: var(--radius-lg);
        }
        
        .deploy-stat-card .stat-icon {
            width: 48px;
            height: 48px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: var(--radius-md);
            font-size: 20px;
        }
        
        .deploy-stat-card .stat-icon.green {
            background: rgba(0,217,181,0.15);
            color: var(--success);
        }
        
        .deploy-stat-card .stat-icon.red {
            background: rgba(255,107,107,0.15);
            color: var(--danger);
        }
        
        .deploy-stat-card .stat-icon.blue {
            background: rgba(54,173,239,0.15);
            color: var(--primary);
        }
        
        .deploy-stat-card .stat-icon.purple {
            background: rgba(123,97,255,0.15);
            color: var(--secondary);
        }
        
        .stat-content {
            display: flex;
            flex-direction: column;
        }
        
        .stat-content .stat-value {
            font-size: 24px;
            font-weight: 700;
        }
        
        .stat-content .stat-label {
            font-size: 13px;
            color: var(--text-secondary);
        }
        
        .version-tag {
            padding: 4px 8px;
            background: var(--bg-elevated);
            border-radius: 4px;
            font-size: 12px;
            font-family: monospace;
        }
        
        .commit-info {
            display: flex;
            flex-direction: column;
            gap: 4px;
        }
        
        .commit-id {
            padding: 2px 6px;
            background: var(--bg-elevated);
            border-radius: 4px;
            font-size: 11px;
            color: var(--text-tertiary);
        }
        
        .commit-msg {
            font-size: 13px;
            color: var(--text-secondary);
            max-width: 200px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }
        
        .table-actions {
            display: flex;
            gap: 4px;
        }
        
        .deploy-log {
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
            padding: 16px;
            max-height: 400px;
            overflow-y: auto;
        }
        
        .log-content {
            font-family: 'SF Mono', 'Fira Code', monospace;
            font-size: 13px;
            line-height: 1.6;
            color: var(--text-secondary);
            white-space: pre-wrap;
        }
        
        .deploy-options {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }
        
        .checkbox-label {
            display: flex;
            align-items: center;
            gap: 8px;
            cursor: pointer;
        }
        
        .checkbox-label input {
            width: 16px;
            height: 16px;
        }
        
        @media (max-width: 1024px) {
            .deploy-stats {
                grid-template-columns: repeat(2, 1fr);
            }
        }
        
        @media (max-width: 768px) {
            .deploy-stats {
                grid-template-columns: 1fr;
            }
        }
    `;
    document.head.appendChild(style);
}

// ===== 开发环境页面 =====

function renderEnvironments() {
    const page = document.getElementById('environmentsPage');
    
    page.innerHTML = `
        <div class="page-header">
            <div class="page-header-left">
                <h2>开发环境</h2>
                <p>管理您的云端开发环境</p>
            </div>
            <div class="page-header-right">
                <div class="search-filter">
                    <input type="text" class="form-input" placeholder="搜索环境..." id="envSearchInput" oninput="filterEnvironments()">
                </div>
                <select class="form-input" id="envStatusFilter" onchange="filterEnvironments()" style="width: 150px;">
                    <option value="all">全部状态</option>
                    <option value="running">运行中</option>
                    <option value="stopped">已停止</option>
                    <option value="creating">创建中</option>
                    <option value="failed">失败</option>
                </select>
                <button class="btn btn-primary" onclick="openModal('createEnvModal')">
                    <i class="fas fa-plus"></i> 创建环境
                </button>
            </div>
        </div>
        
        <!-- 环境统计 -->
        <div class="env-stats">
            <div class="env-stat-item">
                <span class="stat-value">${ENVIRONMENTS.length}</span>
                <span class="stat-label">总环境数</span>
            </div>
            <div class="env-stat-item running">
                <span class="stat-value">${ENVIRONMENTS.filter(e => e.status === 'running').length}</span>
                <span class="stat-label">运行中</span>
            </div>
            <div class="env-stat-item stopped">
                <span class="stat-value">${ENVIRONMENTS.filter(e => e.status === 'stopped').length}</span>
                <span class="stat-label">已停止</span>
            </div>
            <div class="env-stat-item failed">
                <span class="stat-value">${ENVIRONMENTS.filter(e => e.status === 'failed').length}</span>
                <span class="stat-label">失败</span>
            </div>
        </div>
        
        <!-- 环境列表 -->
        <div class="env-list-container" id="envListContainer">
            ${renderEnvList(ENVIRONMENTS)}
        </div>
    `;
    
    addEnvironmentsStyles();
}

function renderEnvList(environments) {
    if (environments.length === 0) {
        return `
            <div class="empty-state">
                <i class="fas fa-server"></i>
                <h3>暂无开发环境</h3>
                <p>创建您的第一个云端开发环境</p>
                <button class="btn btn-primary" onclick="openModal('createEnvModal')">
                    <i class="fas fa-plus"></i> 创建环境
                </button>
            </div>
        `;
    }
    
    return `
        <div class="env-list">
            ${environments.map(env => `
                <div class="env-card" onclick="showEnvDetail('${env.id}')">
                    <div class="env-card-header">
                        <div class="env-card-icon">${env.templateIcon}</div>
                        <div class="env-card-title">
                            <h3>${env.name}</h3>
                            <span class="env-template">${env.templateName}</span>
                        </div>
                        <span class="status-badge ${env.status}">
                            <span class="status-dot"></span>
                            ${getStatusText(env.status)}
                        </span>
                    </div>
                    
                    <div class="env-card-body">
                        <div class="env-resources">
                            <div class="resource-item">
                                <i class="fas fa-microchip"></i>
                                <span>${env.cpu}核 CPU</span>
                            </div>
                            <div class="resource-item">
                                <i class="fas fa-memory"></i>
                                <span>${env.memory}GB 内存</span>
                            </div>
                            <div class="resource-item">
                                <i class="fas fa-hdd"></i>
                                <span>${env.storage}GB 存储</span>
                            </div>
                        </div>
                        
                        ${env.status === 'running' ? `
                            <div class="env-usage">
                                <div class="usage-item">
                                    <span class="usage-label">CPU</span>
                                    <div class="progress-bar">
                                        <div class="progress-fill ${env.cpuUsage > 80 ? 'danger' : env.cpuUsage > 60 ? 'warning' : ''}" 
                                             style="width: ${env.cpuUsage}%"></div>
                                    </div>
                                    <span class="usage-value">${env.cpuUsage}%</span>
                                </div>
                                <div class="usage-item">
                                    <span class="usage-label">内存</span>
                                    <div class="progress-bar">
                                        <div class="progress-fill ${env.memoryUsage > 80 ? 'danger' : env.memoryUsage > 60 ? 'warning' : ''}" 
                                             style="width: ${env.memoryUsage}%"></div>
                                    </div>
                                    <span class="usage-value">${env.memoryUsage}%</span>
                                </div>
                            </div>
                        ` : ''}
                        
                        ${env.domain ? `
                            <div class="env-domain">
                                <i class="fas fa-globe"></i>
                                <a href="https://${env.domain}" target="_blank" onclick="event.stopPropagation()">
                                    ${env.domain}
                                </a>
                                <button class="btn-icon" onclick="event.stopPropagation(); copyToClipboard('https://${env.domain}')" title="复制链接">
                                    <i class="fas fa-copy"></i>
                                </button>
                            </div>
                        ` : ''}
                        
                        ${env.errorMessage ? `
                            <div class="env-error">
                                <i class="fas fa-exclamation-triangle"></i>
                                <span>${env.errorMessage}</span>
                            </div>
                        ` : ''}
                    </div>
                    
                    <div class="env-card-footer">
                        <div class="env-meta-info">
                            <span><i class="fas fa-clock"></i> ${env.createdAt}</span>
                            ${env.collaborators.length > 0 ? `
                                <span><i class="fas fa-users"></i> ${env.collaborators.length}人协作</span>
                            ` : ''}
                        </div>
                        <div class="env-actions" onclick="event.stopPropagation()">
                            ${env.status === 'running' ? `
                                <button class="btn btn-sm btn-outline" onclick="stopEnvironment('${env.id}')" title="停止">
                                    <i class="fas fa-stop"></i>
                                </button>
                                <button class="btn btn-sm btn-outline" onclick="restartEnvironment('${env.id}')" title="重启">
                                    <i class="fas fa-redo"></i>
                                </button>
                            ` : env.status === 'stopped' ? `
                                <button class="btn btn-sm btn-success" onclick="startEnvironment('${env.id}')" title="启动">
                                    <i class="fas fa-play"></i>
                                </button>
                            ` : ''}
                            <button class="btn btn-sm btn-primary" onclick="showEnvDetail('${env.id}')" title="详情">
                                <i class="fas fa-cog"></i>
                            </button>
                            <button class="btn btn-sm btn-danger" onclick="deleteEnvironment('${env.id}')" title="删除">
                                <i class="fas fa-trash"></i>
                            </button>
                        </div>
                    </div>
                </div>
            `).join('')}
        </div>
    `;
}

function filterEnvironments() {
    const searchTerm = document.getElementById('envSearchInput').value.toLowerCase();
    const statusFilter = document.getElementById('envStatusFilter').value;
    
    let filtered = ENVIRONMENTS;
    
    if (searchTerm) {
        filtered = filtered.filter(env => 
            env.name.toLowerCase().includes(searchTerm) ||
            env.templateName.toLowerCase().includes(searchTerm)
        );
    }
    
    if (statusFilter !== 'all') {
        filtered = filtered.filter(env => env.status === statusFilter);
    }
    
    document.getElementById('envListContainer').innerHTML = renderEnvList(filtered);
}

function addEnvironmentsStyles() {
    // 样式已移至全局CSS
}

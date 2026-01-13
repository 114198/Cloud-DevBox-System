// ===== 控制台页面 =====

function renderDashboard() {
    const page = document.getElementById('dashboardPage');
    
    // 计算统计数据
    const runningEnvs = ENVIRONMENTS.filter(e => e.status === 'running').length;
    const totalEnvs = ENVIRONMENTS.length;
    const totalProjects = PROJECTS.length;
    const recentDeployments = DEPLOYMENTS.filter(d => d.status === 'success').length;
    
    page.innerHTML = `
        <!-- 欢迎横幅 -->
        <div class="welcome-banner">
            <div class="welcome-content">
                <h1>欢迎回来，${USER_SETTINGS.profile.displayName}！</h1>
                <p>今天是个写代码的好日子 ☀️</p>
            </div>
            <button class="btn btn-primary btn-lg" onclick="openModal('createEnvModal')">
                <i class="fas fa-plus"></i> 创建新环境
            </button>
        </div>
        
        <!-- 统计卡片 -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-icon blue">
                    <i class="fas fa-server"></i>
                </div>
                <div class="stat-info">
                    <h3>${runningEnvs}/${totalEnvs}</h3>
                    <p>运行中的环境</p>
                </div>
                <div class="stat-trend up">
                    <i class="fas fa-arrow-up"></i> 2
                </div>
            </div>
            <div class="stat-card">
                <div class="stat-icon green">
                    <i class="fas fa-folder"></i>
                </div>
                <div class="stat-info">
                    <h3>${totalProjects}</h3>
                    <p>项目总数</p>
                </div>
                <div class="stat-trend up">
                    <i class="fas fa-arrow-up"></i> 1
                </div>
            </div>
            <div class="stat-card">
                <div class="stat-icon yellow">
                    <i class="fas fa-rocket"></i>
                </div>
                <div class="stat-info">
                    <h3>${recentDeployments}</h3>
                    <p>本月部署次数</p>
                </div>
                <div class="stat-trend up">
                    <i class="fas fa-arrow-up"></i> 15%
                </div>
            </div>
            <div class="stat-card">
                <div class="stat-icon purple">
                    <i class="fas fa-clock"></i>
                </div>
                <div class="stat-info">
                    <h3>128h</h3>
                    <p>本月使用时长</p>
                </div>
                <div class="stat-trend down">
                    <i class="fas fa-arrow-down"></i> 5%
                </div>
            </div>
        </div>
        
        <!-- 主要内容区 -->
        <div class="dashboard-grid">
            <!-- 最近环境 -->
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">最近使用的环境</h3>
                    <a href="#environments" class="btn btn-outline btn-sm" onclick="loadPage('environments')">
                        查看全部 <i class="fas fa-arrow-right"></i>
                    </a>
                </div>
                <div class="card-body">
                    <div class="env-list">
                        ${ENVIRONMENTS.slice(0, 3).map(env => `
                            <div class="env-item" onclick="showEnvDetail('${env.id}')">
                                <div class="env-icon">${env.templateIcon}</div>
                                <div class="env-info">
                                    <div class="env-name">
                                        ${env.name}
                                        <span class="status-badge ${env.status}">
                                            <span class="status-dot"></span>
                                            ${getStatusText(env.status)}
                                        </span>
                                    </div>
                                    <div class="env-meta">
                                        <span><i class="fas fa-microchip"></i> ${env.cpu}核</span>
                                        <span><i class="fas fa-memory"></i> ${env.memory}GB</span>
                                        <span><i class="fas fa-clock"></i> ${env.lastAccessed}</span>
                                    </div>
                                </div>
                                <div class="env-actions">
                                    ${env.status === 'running' ? `
                                        <button class="btn btn-sm btn-outline" onclick="event.stopPropagation(); stopEnvironment('${env.id}')" title="停止">
                                            <i class="fas fa-stop"></i>
                                        </button>
                                    ` : env.status === 'stopped' ? `
                                        <button class="btn btn-sm btn-success" onclick="event.stopPropagation(); startEnvironment('${env.id}')" title="启动">
                                            <i class="fas fa-play"></i>
                                        </button>
                                    ` : ''}
                                    <button class="btn btn-sm btn-primary" onclick="event.stopPropagation(); showEnvDetail('${env.id}')" title="详情">
                                        <i class="fas fa-external-link-alt"></i>
                                    </button>
                                </div>
                            </div>
                        `).join('')}
                    </div>
                </div>
            </div>
            
            <!-- 快速操作 -->
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">快速操作</h3>
                </div>
                <div class="card-body">
                    <div class="quick-actions">
                        <button class="quick-action-btn" onclick="openModal('createEnvModal')">
                            <div class="quick-action-icon blue">
                                <i class="fas fa-plus"></i>
                            </div>
                            <span>创建环境</span>
                        </button>
                        <button class="quick-action-btn" onclick="loadPage('templates')">
                            <div class="quick-action-icon green">
                                <i class="fas fa-layer-group"></i>
                            </div>
                            <span>浏览模板</span>
                        </button>
                        <button class="quick-action-btn" onclick="loadPage('projects')">
                            <div class="quick-action-icon yellow">
                                <i class="fas fa-folder-plus"></i>
                            </div>
                            <span>新建项目</span>
                        </button>
                        <button class="quick-action-btn" onclick="loadPage('collaboration')">
                            <div class="quick-action-icon purple">
                                <i class="fas fa-users"></i>
                            </div>
                            <span>邀请协作</span>
                        </button>
                    </div>
                </div>
            </div>
            
            <!-- 最近部署 -->
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">最近部署</h3>
                    <a href="#deployments" class="btn btn-outline btn-sm" onclick="loadPage('deployments')">
                        查看全部 <i class="fas fa-arrow-right"></i>
                    </a>
                </div>
                <div class="card-body">
                    <div class="deployment-list">
                        ${DEPLOYMENTS.slice(0, 4).map(deploy => `
                            <div class="deployment-item">
                                <div class="deployment-status ${deploy.status}">
                                    <i class="fas ${deploy.status === 'success' ? 'fa-check-circle' : 'fa-times-circle'}"></i>
                                </div>
                                <div class="deployment-info">
                                    <div class="deployment-name">${deploy.envName} <span class="version">${deploy.version}</span></div>
                                    <div class="deployment-meta">
                                        <span>${deploy.commitMessage}</span>
                                    </div>
                                </div>
                                <div class="deployment-time">
                                    ${formatRelativeTime(deploy.startTime)}
                                </div>
                            </div>
                        `).join('')}
                    </div>
                </div>
            </div>
            
            <!-- 系统通知 -->
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">系统通知</h3>
                    <button class="btn btn-outline btn-sm">
                        全部已读
                    </button>
                </div>
                <div class="card-body">
                    <div class="notification-list">
                        ${NOTIFICATIONS.map(notif => `
                            <div class="notification-item ${notif.read ? 'read' : ''}">
                                <div class="notification-icon ${notif.type}">
                                    <i class="fas ${notif.type === 'success' ? 'fa-check' : notif.type === 'warning' ? 'fa-exclamation' : 'fa-info'}"></i>
                                </div>
                                <div class="notification-content">
                                    <div class="notification-title">${notif.title}</div>
                                    <div class="notification-message">${notif.message}</div>
                                </div>
                                <div class="notification-time">${notif.time}</div>
                            </div>
                        `).join('')}
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // 添加仪表板特定样式
    addDashboardStyles();
}

function addDashboardStyles() {
    // 样式已移至全局CSS
}

// ===== 全局应用逻辑 =====

// 当前状态
let currentPage = 'dashboard';
let currentStep = 1;
let selectedTemplate = null;
let currentTheme = 'dark';
let envConfig = {
    name: '',
    cpu: '1',
    memory: '2',
    storage: '20',
    gitRepo: '',
    envVars: [],
    ports: [{ container: 3000, external: 80 }]
};

// 初始化应用
document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    initSidebar();
    initNavigation();
    initModals();
    loadPage('dashboard');
});

// 初始化主题
function initTheme() {
    // 从 localStorage 读取保存的主题
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme) {
        currentTheme = savedTheme;
    } else {
        // 检测系统主题偏好
        const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        currentTheme = prefersDark ? 'dark' : 'light';
    }
    
    applyTheme(currentTheme);
    
    // 绑定主题切换按钮
    const themeToggle = document.getElementById('themeToggle');
    if (themeToggle) {
        themeToggle.addEventListener('click', toggleTheme);
    }
    
    // 监听系统主题变化
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
        if (!localStorage.getItem('theme')) {
            currentTheme = e.matches ? 'dark' : 'light';
            applyTheme(currentTheme);
        }
    });
}

// 切换主题
function toggleTheme() {
    currentTheme = currentTheme === 'dark' ? 'light' : 'dark';
    applyTheme(currentTheme);
    localStorage.setItem('theme', currentTheme);
}

// 应用主题
function applyTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    
    // 更新切换按钮图标
    const themeToggle = document.getElementById('themeToggle');
    if (themeToggle) {
        const icon = themeToggle.querySelector('i');
        if (icon) {
            icon.className = theme === 'dark' ? 'fas fa-moon' : 'fas fa-sun';
        }
        themeToggle.title = theme === 'dark' ? '切换到浅色模式' : '切换到深色模式';
    }
}

// 初始化侧边栏
function initSidebar() {
    const sidebar = document.getElementById('sidebar');
    const toggle = document.getElementById('sidebarToggle');
    
    toggle.addEventListener('click', () => {
        sidebar.classList.toggle('collapsed');
    });
}

// 初始化导航
function initNavigation() {
    const navItems = document.querySelectorAll('.sidebar-nav li');
    
    navItems.forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const page = item.dataset.page;
            
            // 更新活动状态
            navItems.forEach(i => i.classList.remove('active'));
            item.classList.add('active');
            
            // 加载页面
            loadPage(page);
        });
    });
}

// 加载页面
function loadPage(page) {
    currentPage = page;
    
    // 隐藏所有页面
    document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
    
    // 显示目标页面
    const targetPage = document.getElementById(`${page}Page`);
    if (targetPage) {
        targetPage.classList.add('active');
    }
    
    // 更新面包屑
    const pageTitle = document.getElementById('currentPageTitle');
    const titles = {
        dashboard: '控制台',
        environments: '开发环境',
        templates: '环境模板',
        projects: '项目管理',
        deployments: '部署管理',
        collaboration: '协作开发',
        monitoring: '监控中心',
        billing: '费用账单',
        settings: '系统设置'
    };
    pageTitle.textContent = titles[page] || page;
    
    // 调用页面初始化函数
    switch (page) {
        case 'dashboard':
            if (typeof renderDashboard === 'function') renderDashboard();
            break;
        case 'environments':
            if (typeof renderEnvironments === 'function') renderEnvironments();
            break;
        case 'templates':
            if (typeof renderTemplates === 'function') renderTemplates();
            break;
        case 'projects':
            if (typeof renderProjects === 'function') renderProjects();
            break;
        case 'deployments':
            if (typeof renderDeployments === 'function') renderDeployments();
            break;
        case 'collaboration':
            if (typeof renderCollaboration === 'function') renderCollaboration();
            break;
        case 'monitoring':
            if (typeof renderMonitoring === 'function') renderMonitoring();
            break;
        case 'billing':
            if (typeof renderBilling === 'function') renderBilling();
            break;
        case 'settings':
            if (typeof renderSettings === 'function') renderSettings();
            break;
    }
}

// 初始化模态框
function initModals() {
    // 点击遮罩关闭
    document.querySelectorAll('.modal-overlay').forEach(overlay => {
        overlay.addEventListener('click', () => {
            const modal = overlay.closest('.modal');
            if (modal) {
                modal.classList.remove('active');
            }
        });
    });
    
    // 初始化模板选择
    initTemplateSelection();
    
    // 初始化资源选择
    initResourceSelection();
}

// 打开模态框
function openModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.add('active');
        
        // 如果是创建环境模态框，重置状态
        if (modalId === 'createEnvModal') {
            resetCreateEnvModal();
        }
    }
}

// 关闭模态框
function closeModal(modalId) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.classList.remove('active');
    }
}

// 重置创建环境模态框
function resetCreateEnvModal() {
    currentStep = 1;
    selectedTemplate = null;
    envConfig = {
        name: '',
        cpu: '1',
        memory: '2',
        storage: '20',
        gitRepo: '',
        envVars: [],
        ports: [{ container: 3000, external: 80 }]
    };
    
    updateStepIndicator();
    showStepPanel(1);
    updateStepButtons();
    
    // 重置表单
    document.getElementById('envName').value = '';
    document.getElementById('gitRepo').value = '';
    
    // 重置模板选择
    document.querySelectorAll('.template-card').forEach(card => {
        card.classList.remove('selected');
    });
    
    // 重置资源选择
    document.querySelectorAll('.resource-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelector('#cpuOptions .resource-btn[data-value="1"]').classList.add('active');
    document.querySelector('#memoryOptions .resource-btn[data-value="2"]').classList.add('active');
    document.querySelector('#storageOptions .resource-btn[data-value="20"]').classList.add('active');
}

// 初始化模板选择
function initTemplateSelection() {
    const templateGrid = document.getElementById('templateGrid');
    if (!templateGrid) return;
    
    // 渲染模板
    renderTemplateGrid('all');
    
    // 分类按钮事件
    document.querySelectorAll('.category-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.category-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            renderTemplateGrid(btn.dataset.category);
        });
    });
}

// 渲染模板网格
function renderTemplateGrid(category) {
    const templateGrid = document.getElementById('templateGrid');
    if (!templateGrid) return;
    
    const filteredTemplates = category === 'all' 
        ? TEMPLATES 
        : TEMPLATES.filter(t => t.category === category);
    
    templateGrid.innerHTML = filteredTemplates.map(template => `
        <div class="template-card ${selectedTemplate === template.id ? 'selected' : ''}" 
             data-id="${template.id}" onclick="selectTemplate('${template.id}')">
            <div class="icon">${template.icon}</div>
            <div class="name">${template.name}</div>
            <div class="version">${template.version}</div>
        </div>
    `).join('');
}

// 选择模板
function selectTemplate(templateId) {
    selectedTemplate = templateId;
    
    document.querySelectorAll('.template-card').forEach(card => {
        card.classList.toggle('selected', card.dataset.id === templateId);
    });
}

// 初始化资源选择
function initResourceSelection() {
    ['cpuOptions', 'memoryOptions', 'storageOptions'].forEach(optionsId => {
        const container = document.getElementById(optionsId);
        if (!container) return;
        
        container.querySelectorAll('.resource-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                container.querySelectorAll('.resource-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                
                const value = btn.dataset.value;
                if (optionsId === 'cpuOptions') envConfig.cpu = value;
                if (optionsId === 'memoryOptions') envConfig.memory = value;
                if (optionsId === 'storageOptions') envConfig.storage = value;
            });
        });
    });
}

// 下一步
function nextStep() {
    if (currentStep === 1 && !selectedTemplate) {
        showToast('请选择一个模板', 'warning');
        return;
    }
    
    if (currentStep === 2) {
        const envName = document.getElementById('envName').value.trim();
        if (!envName) {
            showToast('请输入环境名称', 'warning');
            return;
        }
        envConfig.name = envName;
    }
    
    if (currentStep === 3) {
        envConfig.gitRepo = document.getElementById('gitRepo').value.trim();
    }
    
    if (currentStep < 4) {
        currentStep++;
        updateStepIndicator();
        showStepPanel(currentStep);
        updateStepButtons();
        
        // 更新确认摘要
        if (currentStep === 4) {
            updateSummary();
        }
    }
}

// 上一步
function prevStep() {
    if (currentStep > 1) {
        currentStep--;
        updateStepIndicator();
        showStepPanel(currentStep);
        updateStepButtons();
    }
}

// 更新步骤指示器
function updateStepIndicator() {
    document.querySelectorAll('.steps-indicator .step').forEach((step, index) => {
        const stepNum = index + 1;
        step.classList.remove('active', 'completed');
        
        if (stepNum < currentStep) {
            step.classList.add('completed');
        } else if (stepNum === currentStep) {
            step.classList.add('active');
        }
    });
}

// 显示步骤面板
function showStepPanel(step) {
    document.querySelectorAll('.step-panel').forEach(panel => {
        panel.classList.toggle('active', parseInt(panel.dataset.step) === step);
    });
}

// 更新步骤按钮
function updateStepButtons() {
    const prevBtn = document.getElementById('prevStepBtn');
    const nextBtn = document.getElementById('nextStepBtn');
    const createBtn = document.getElementById('createEnvBtn');
    
    prevBtn.style.display = currentStep > 1 ? 'inline-flex' : 'none';
    nextBtn.style.display = currentStep < 4 ? 'inline-flex' : 'none';
    createBtn.style.display = currentStep === 4 ? 'inline-flex' : 'none';
}

// 更新确认摘要
function updateSummary() {
    const template = TEMPLATES.find(t => t.id === selectedTemplate);
    
    document.getElementById('summaryTemplate').textContent = template ? template.name : '-';
    document.getElementById('summaryName').textContent = envConfig.name || '-';
    document.getElementById('summaryCpu').textContent = envConfig.cpu + '核';
    document.getElementById('summaryMemory').textContent = envConfig.memory + 'GB';
    document.getElementById('summaryStorage').textContent = envConfig.storage + 'GB';
    
    // 计算价格
    const cpuPrice = parseFloat(envConfig.cpu) * 0.1;
    const memoryPrice = parseFloat(envConfig.memory) * 0.05;
    const storagePrice = parseFloat(envConfig.storage) * 0.01;
    const totalPrice = (cpuPrice + memoryPrice + storagePrice).toFixed(2);
    
    document.getElementById('summaryPrice').textContent = `¥${totalPrice}/小时`;
}

// 创建环境
function createEnvironment() {
    showToast('正在创建环境...', 'info');
    
    // 模拟创建过程
    setTimeout(() => {
        const template = TEMPLATES.find(t => t.id === selectedTemplate);
        
        // 添加新环境到列表
        const newEnv = {
            id: 'env-' + Date.now(),
            name: envConfig.name,
            template: selectedTemplate,
            templateName: template.name,
            templateIcon: template.icon,
            status: 'creating',
            cpu: envConfig.cpu,
            memory: envConfig.memory,
            storage: envConfig.storage,
            domain: '',
            sshHost: '',
            sshPort: 0,
            createdAt: new Date().toLocaleString(),
            lastAccessed: '-',
            uptime: '-',
            cpuUsage: 0,
            memoryUsage: 0,
            storageUsage: 0,
            gitRepo: envConfig.gitRepo,
            collaborators: []
        };
        
        ENVIRONMENTS.unshift(newEnv);
        
        closeModal('createEnvModal');
        showToast('环境创建成功！', 'success');
        
        // 刷新环境列表
        if (currentPage === 'environments') {
            renderEnvironments();
        } else if (currentPage === 'dashboard') {
            renderDashboard();
        }
        
        // 模拟环境启动
        setTimeout(() => {
            newEnv.status = 'running';
            newEnv.domain = `${envConfig.name}.devbox.io`;
            newEnv.sshHost = 'ssh.devbox.io';
            newEnv.sshPort = 22000 + ENVIRONMENTS.length;
            
            if (currentPage === 'environments') {
                renderEnvironments();
            } else if (currentPage === 'dashboard') {
                renderDashboard();
            }
            
            showToast(`环境 ${envConfig.name} 已启动`, 'success');
        }, 3000);
    }, 1500);
}

// 添加环境变量
function addEnvVar() {
    const container = document.getElementById('envVars');
    const row = document.createElement('div');
    row.className = 'env-var-row';
    row.innerHTML = `
        <input type="text" placeholder="KEY" class="form-input">
        <input type="text" placeholder="VALUE" class="form-input">
        <button class="btn-icon" onclick="removeEnvVar(this)"><i class="fas fa-trash"></i></button>
    `;
    container.appendChild(row);
}

// 移除环境变量
function removeEnvVar(btn) {
    const row = btn.closest('.env-var-row');
    if (document.querySelectorAll('.env-var-row').length > 1) {
        row.remove();
    }
}

// 添加端口映射
function addPort() {
    const container = document.getElementById('portMappings');
    const row = document.createElement('div');
    row.className = 'port-row';
    row.innerHTML = `
        <input type="number" placeholder="容器端口" class="form-input">
        <span>→</span>
        <input type="number" placeholder="外部端口" class="form-input">
        <button class="btn-icon" onclick="removePort(this)"><i class="fas fa-trash"></i></button>
    `;
    container.appendChild(row);
}

// 移除端口映射
function removePort(btn) {
    const row = btn.closest('.port-row');
    if (document.querySelectorAll('.port-row').length > 1) {
        row.remove();
    }
}

// 显示Toast通知
function showToast(message, type = 'info') {
    const container = document.getElementById('toastContainer');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    
    const icons = {
        success: 'fa-check-circle',
        error: 'fa-times-circle',
        warning: 'fa-exclamation-triangle',
        info: 'fa-info-circle'
    };
    
    toast.innerHTML = `
        <i class="fas ${icons[type]} toast-icon"></i>
        <span class="toast-message">${message}</span>
        <i class="fas fa-times toast-close" onclick="this.parentElement.remove()"></i>
    `;
    
    container.appendChild(toast);
    
    // 自动移除
    setTimeout(() => {
        toast.remove();
    }, 5000);
}

// 环境操作
function startEnvironment(envId) {
    const env = ENVIRONMENTS.find(e => e.id === envId);
    if (env) {
        env.status = 'running';
        showToast(`环境 ${env.name} 已启动`, 'success');
        renderEnvironments();
    }
}

function stopEnvironment(envId) {
    const env = ENVIRONMENTS.find(e => e.id === envId);
    if (env) {
        env.status = 'stopped';
        showToast(`环境 ${env.name} 已停止`, 'info');
        renderEnvironments();
    }
}

function restartEnvironment(envId) {
    const env = ENVIRONMENTS.find(e => e.id === envId);
    if (env) {
        env.status = 'creating';
        showToast(`正在重启环境 ${env.name}...`, 'info');
        renderEnvironments();
        
        setTimeout(() => {
            env.status = 'running';
            showToast(`环境 ${env.name} 已重启`, 'success');
            renderEnvironments();
        }, 2000);
    }
}

function deleteEnvironment(envId) {
    if (confirm('确定要删除这个环境吗？此操作不可恢复。')) {
        const index = ENVIRONMENTS.findIndex(e => e.id === envId);
        if (index > -1) {
            const env = ENVIRONMENTS[index];
            ENVIRONMENTS.splice(index, 1);
            showToast(`环境 ${env.name} 已删除`, 'success');
            renderEnvironments();
        }
    }
}

// 显示环境详情
function showEnvDetail(envId) {
    const env = ENVIRONMENTS.find(e => e.id === envId);
    if (!env) return;
    
    const content = document.getElementById('envDetailContent');
    content.innerHTML = `
        <div class="env-detail">
            <div class="env-detail-header">
                <div class="env-detail-icon">${env.templateIcon}</div>
                <div class="env-detail-info">
                    <h2>${env.name}</h2>
                    <p>${env.templateName} · ${env.cpu}核 · ${env.memory}GB内存 · ${env.storage}GB存储</p>
                </div>
                <div class="env-detail-status">
                    <span class="status-badge ${env.status}">
                        <span class="status-dot"></span>
                        ${getStatusText(env.status)}
                    </span>
                </div>
            </div>
            
            <div class="tabs">
                <button class="tab-btn active" onclick="switchEnvTab('overview')">概览</button>
                <button class="tab-btn" onclick="switchEnvTab('connection')">连接信息</button>
                <button class="tab-btn" onclick="switchEnvTab('resources')">资源监控</button>
                <button class="tab-btn" onclick="switchEnvTab('logs')">日志</button>
            </div>
            
            <div class="tab-content active" id="overviewTab">
                <div class="detail-grid">
                    <div class="detail-item">
                        <label>环境ID</label>
                        <span>${env.id}</span>
                    </div>
                    <div class="detail-item">
                        <label>创建时间</label>
                        <span>${env.createdAt}</span>
                    </div>
                    <div class="detail-item">
                        <label>最后访问</label>
                        <span>${env.lastAccessed}</span>
                    </div>
                    <div class="detail-item">
                        <label>运行时长</label>
                        <span>${env.uptime}</span>
                    </div>
                    <div class="detail-item">
                        <label>外网域名</label>
                        <span>${env.domain || '-'}</span>
                        ${env.domain ? `<button class="btn btn-sm btn-outline" onclick="copyToClipboard('https://${env.domain}')"><i class="fas fa-copy"></i></button>` : ''}
                    </div>
                    <div class="detail-item">
                        <label>Git仓库</label>
                        <span>${env.gitRepo || '未配置'}</span>
                    </div>
                </div>
            </div>
            
            <div class="tab-content" id="connectionTab">
                <div class="connection-info">
                    <h4>SSH 连接</h4>
                    <div class="code-block">
                        <code>ssh devbox@${env.sshHost} -p ${env.sshPort}</code>
                        <button class="btn-icon" onclick="copyToClipboard('ssh devbox@${env.sshHost} -p ${env.sshPort}')">
                            <i class="fas fa-copy"></i>
                        </button>
                    </div>
                    
                    <h4>VSCode Remote SSH 配置</h4>
                    <div class="code-block">
                        <pre>Host ${env.name}
    HostName ${env.sshHost}
    Port ${env.sshPort}
    User devbox
    StrictHostKeyChecking no</pre>
                        <button class="btn-icon" onclick="copySSHConfig('${env.name}', '${env.sshHost}', ${env.sshPort})">
                            <i class="fas fa-copy"></i>
                        </button>
                    </div>
                    
                    <div class="connection-buttons">
                        <button class="btn btn-primary" onclick="openInVSCode('${env.id}')">
                            <i class="fas fa-external-link-alt"></i> 在 VSCode 中打开
                        </button>
                        <button class="btn btn-outline" onclick="downloadSSHKey('${env.id}')">
                            <i class="fas fa-key"></i> 下载 SSH 密钥
                        </button>
                    </div>
                </div>
            </div>
            
            <div class="tab-content" id="resourcesTab">
                <div class="resource-charts">
                    <div class="resource-chart">
                        <h4>CPU 使用率</h4>
                        <div class="progress-bar">
                            <div class="progress-fill ${env.cpuUsage > 80 ? 'danger' : env.cpuUsage > 60 ? 'warning' : ''}" 
                                 style="width: ${env.cpuUsage}%"></div>
                        </div>
                        <span>${env.cpuUsage}%</span>
                    </div>
                    <div class="resource-chart">
                        <h4>内存使用率</h4>
                        <div class="progress-bar">
                            <div class="progress-fill ${env.memoryUsage > 80 ? 'danger' : env.memoryUsage > 60 ? 'warning' : ''}" 
                                 style="width: ${env.memoryUsage}%"></div>
                        </div>
                        <span>${env.memoryUsage}%</span>
                    </div>
                    <div class="resource-chart">
                        <h4>存储使用率</h4>
                        <div class="progress-bar">
                            <div class="progress-fill ${env.storageUsage > 80 ? 'danger' : env.storageUsage > 60 ? 'warning' : ''}" 
                                 style="width: ${env.storageUsage}%"></div>
                        </div>
                        <span>${env.storageUsage}%</span>
                    </div>
                </div>
            </div>
            
            <div class="tab-content" id="logsTab">
                <div class="log-viewer">
                    <div class="log-toolbar">
                        <select class="form-input" style="width: auto;">
                            <option>全部日志</option>
                            <option>应用日志</option>
                            <option>系统日志</option>
                            <option>错误日志</option>
                        </select>
                        <button class="btn btn-outline btn-sm">
                            <i class="fas fa-download"></i> 下载日志
                        </button>
                    </div>
                    <div class="log-content">
                        <pre>[2026-01-09 10:30:15] INFO: Application started on port 3000
[2026-01-09 10:30:16] INFO: Connected to database
[2026-01-09 10:30:17] INFO: Server is ready to accept connections
[2026-01-09 10:35:22] INFO: GET /api/users - 200 OK (15ms)
[2026-01-09 10:35:45] INFO: POST /api/login - 200 OK (42ms)
[2026-01-09 10:40:12] WARN: High memory usage detected (75%)
[2026-01-09 10:45:33] INFO: GET /api/products - 200 OK (28ms)</pre>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    openModal('envDetailModal');
}

// 切换环境详情标签
function switchEnvTab(tabName) {
    document.querySelectorAll('#envDetailContent .tab-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelectorAll('#envDetailContent .tab-content').forEach(content => {
        content.classList.remove('active');
    });
    
    event.target.classList.add('active');
    document.getElementById(`${tabName}Tab`).classList.add('active');
}

// 获取状态文本
function getStatusText(status) {
    const texts = {
        running: '运行中',
        stopped: '已停止',
        creating: '创建中',
        failed: '创建失败',
        suspended: '已挂起'
    };
    return texts[status] || status;
}

// 复制到剪贴板
function copyToClipboard(text) {
    navigator.clipboard.writeText(text).then(() => {
        showToast('已复制到剪贴板', 'success');
    });
}

// 复制SSH配置
function copySSHConfig(name, host, port) {
    const config = `Host ${name}
    HostName ${host}
    Port ${port}
    User devbox
    StrictHostKeyChecking no`;
    copyToClipboard(config);
}

// 在VSCode中打开
function openInVSCode(envId) {
    showToast('正在启动 VSCode...', 'info');
    // 实际实现中会调用 vscode:// 协议
}

// 下载SSH密钥
function downloadSSHKey(envId) {
    showToast('正在下载 SSH 密钥...', 'info');
    // 实际实现中会下载密钥文件
}

// 格式化日期
function formatDate(dateStr) {
    const date = new Date(dateStr);
    return date.toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// 格式化相对时间
function formatRelativeTime(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now - date;
    
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (minutes < 1) return '刚刚';
    if (minutes < 60) return `${minutes}分钟前`;
    if (hours < 24) return `${hours}小时前`;
    return `${days}天前`;
}

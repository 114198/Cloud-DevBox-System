// ===== 项目管理页面 =====

function renderProjects() {
    const page = document.getElementById('projectsPage');
    
    page.innerHTML = `
        <div class="page-header">
            <div class="page-header-left">
                <h2>项目管理</h2>
                <p>管理您的开发项目和团队协作</p>
            </div>
            <div class="page-header-right">
                <button class="btn btn-primary" onclick="openCreateProjectModal()">
                    <i class="fas fa-plus"></i> 新建项目
                </button>
            </div>
        </div>
        
        <!-- 项目列表 -->
        <div class="projects-grid">
            ${PROJECTS.map(project => `
                <div class="project-card" onclick="showProjectDetail('${project.id}')">
                    <div class="project-card-header">
                        <div class="project-icon">
                            <i class="fas fa-folder"></i>
                        </div>
                        <div class="project-info">
                            <h3>${project.name}</h3>
                            <p>${project.description}</p>
                        </div>
                    </div>
                    
                    <div class="project-card-body">
                        <div class="project-stats">
                            <div class="project-stat">
                                <i class="fas fa-server"></i>
                                <span>${project.environments.length} 个环境</span>
                            </div>
                            <div class="project-stat">
                                <i class="fas fa-users"></i>
                                <span>${project.members.length} 名成员</span>
                            </div>
                        </div>
                        
                        ${project.gitRepo ? `
                            <div class="project-repo">
                                <i class="fab fa-github"></i>
                                <a href="${project.gitRepo}" target="_blank" onclick="event.stopPropagation()">
                                    ${project.gitRepo.replace('https://github.com/', '')}
                                </a>
                            </div>
                        ` : ''}
                        
                        <div class="project-members">
                            ${project.members.slice(0, 4).map(member => `
                                <img src="${member.avatar}" alt="${member.name}" title="${member.name} (${getRoleName(member.role)})" class="member-avatar">
                            `).join('')}
                            ${project.members.length > 4 ? `
                                <span class="member-more">+${project.members.length - 4}</span>
                            ` : ''}
                        </div>
                    </div>
                    
                    <div class="project-card-footer">
                        <span class="project-date">
                            <i class="fas fa-clock"></i> 更新于 ${project.updatedAt}
                        </span>
                        <div class="project-actions" onclick="event.stopPropagation()">
                            <button class="btn-icon" onclick="editProject('${project.id}')" title="编辑">
                                <i class="fas fa-edit"></i>
                            </button>
                            <button class="btn-icon" onclick="deleteProject('${project.id}')" title="删除">
                                <i class="fas fa-trash"></i>
                            </button>
                        </div>
                    </div>
                </div>
            `).join('')}
            
            <!-- 新建项目卡片 -->
            <div class="project-card new-project" onclick="openCreateProjectModal()">
                <div class="new-project-content">
                    <i class="fas fa-plus-circle"></i>
                    <span>新建项目</span>
                </div>
            </div>
        </div>
    `;
    
    addProjectsStyles();
}

function getRoleName(role) {
    const roles = {
        'owner': '所有者',
        'editor': '编辑者',
        'viewer': '查看者'
    };
    return roles[role] || role;
}

function openCreateProjectModal() {
    // 创建模态框
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'createProjectModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('createProjectModal')"></div>
        <div class="modal-content">
            <div class="modal-header">
                <h3>新建项目</h3>
                <button class="modal-close" onclick="closeModal('createProjectModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="form-group">
                    <label>项目名称</label>
                    <input type="text" id="projectName" class="form-input" placeholder="输入项目名称">
                </div>
                <div class="form-group">
                    <label>项目描述</label>
                    <textarea id="projectDesc" class="form-input" rows="3" placeholder="输入项目描述"></textarea>
                </div>
                <div class="form-group">
                    <label>Git 仓库 (可选)</label>
                    <input type="text" id="projectGit" class="form-input" placeholder="https://github.com/username/repo.git">
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-outline" onclick="closeModal('createProjectModal')">取消</button>
                <button class="btn btn-primary" onclick="createProject()">创建项目</button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function createProject() {
    const name = document.getElementById('projectName').value.trim();
    const desc = document.getElementById('projectDesc').value.trim();
    const git = document.getElementById('projectGit').value.trim();
    
    if (!name) {
        showToast('请输入项目名称', 'warning');
        return;
    }
    
    const newProject = {
        id: 'proj-' + Date.now(),
        name: name,
        description: desc || '暂无描述',
        environments: [],
        members: [
            { name: USER_SETTINGS.profile.displayName, role: 'owner', avatar: USER_SETTINGS.profile.avatar }
        ],
        gitRepo: git,
        createdAt: new Date().toISOString().split('T')[0],
        updatedAt: new Date().toISOString().split('T')[0]
    };
    
    PROJECTS.push(newProject);
    closeModal('createProjectModal');
    showToast('项目创建成功', 'success');
    renderProjects();
}

function showProjectDetail(projectId) {
    const project = PROJECTS.find(p => p.id === projectId);
    if (!project) return;
    
    const projectEnvs = ENVIRONMENTS.filter(e => project.environments.includes(e.id));
    
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'projectDetailModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('projectDetailModal')"></div>
        <div class="modal-content modal-xl">
            <div class="modal-header">
                <h3>${project.name}</h3>
                <button class="modal-close" onclick="closeModal('projectDetailModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="tabs">
                    <button class="tab-btn active" onclick="switchProjectTab('overview')">概览</button>
                    <button class="tab-btn" onclick="switchProjectTab('environments')">环境</button>
                    <button class="tab-btn" onclick="switchProjectTab('members')">成员</button>
                    <button class="tab-btn" onclick="switchProjectTab('settings')">设置</button>
                </div>
                
                <div class="tab-content active" id="projectOverviewTab">
                    <div class="project-detail-info">
                        <p>${project.description}</p>
                        ${project.gitRepo ? `
                            <div class="project-repo-link">
                                <i class="fab fa-github"></i>
                                <a href="${project.gitRepo}" target="_blank">${project.gitRepo}</a>
                            </div>
                        ` : ''}
                        <div class="project-meta">
                            <span><i class="fas fa-calendar"></i> 创建于 ${project.createdAt}</span>
                            <span><i class="fas fa-clock"></i> 更新于 ${project.updatedAt}</span>
                        </div>
                    </div>
                </div>
                
                <div class="tab-content" id="projectEnvironmentsTab">
                    ${projectEnvs.length > 0 ? `
                        <div class="env-list">
                            ${projectEnvs.map(env => `
                                <div class="env-item">
                                    <div class="env-icon">${env.templateIcon}</div>
                                    <div class="env-info">
                                        <div class="env-name">${env.name}</div>
                                        <div class="env-meta">
                                            <span>${env.templateName}</span>
                                            <span>${env.cpu}核 / ${env.memory}GB</span>
                                        </div>
                                    </div>
                                    <span class="status-badge ${env.status}">
                                        <span class="status-dot"></span>
                                        ${getStatusText(env.status)}
                                    </span>
                                </div>
                            `).join('')}
                        </div>
                    ` : `
                        <div class="empty-state">
                            <i class="fas fa-server"></i>
                            <p>暂无关联环境</p>
                            <button class="btn btn-primary btn-sm" onclick="closeModal('projectDetailModal'); openModal('createEnvModal')">
                                创建环境
                            </button>
                        </div>
                    `}
                </div>
                
                <div class="tab-content" id="projectMembersTab">
                    <div class="members-list">
                        ${project.members.map(member => `
                            <div class="member-item">
                                <img src="${member.avatar}" alt="${member.name}" class="member-avatar-lg">
                                <div class="member-info">
                                    <span class="member-name">${member.name}</span>
                                    <span class="member-role">${getRoleName(member.role)}</span>
                                </div>
                                ${member.role !== 'owner' ? `
                                    <button class="btn-icon" onclick="removeMember('${project.id}', '${member.name}')" title="移除">
                                        <i class="fas fa-times"></i>
                                    </button>
                                ` : ''}
                            </div>
                        `).join('')}
                    </div>
                    <button class="btn btn-outline btn-sm" onclick="inviteMember('${project.id}')">
                        <i class="fas fa-user-plus"></i> 邀请成员
                    </button>
                </div>
                
                <div class="tab-content" id="projectSettingsTab">
                    <div class="form-group">
                        <label>项目名称</label>
                        <input type="text" class="form-input" value="${project.name}">
                    </div>
                    <div class="form-group">
                        <label>项目描述</label>
                        <textarea class="form-input" rows="3">${project.description}</textarea>
                    </div>
                    <div class="form-group">
                        <label>Git 仓库</label>
                        <input type="text" class="form-input" value="${project.gitRepo || ''}">
                    </div>
                    <div class="danger-zone">
                        <h4>危险操作</h4>
                        <button class="btn btn-danger" onclick="deleteProject('${project.id}'); closeModal('projectDetailModal')">
                            <i class="fas fa-trash"></i> 删除项目
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function switchProjectTab(tabName) {
    document.querySelectorAll('#projectDetailModal .tab-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelectorAll('#projectDetailModal .tab-content').forEach(content => {
        content.classList.remove('active');
    });
    
    event.target.classList.add('active');
    document.getElementById(`project${tabName.charAt(0).toUpperCase() + tabName.slice(1)}Tab`).classList.add('active');
}

function editProject(projectId) {
    showProjectDetail(projectId);
    // 切换到设置标签
    setTimeout(() => {
        const settingsBtn = document.querySelector('#projectDetailModal .tab-btn:last-child');
        if (settingsBtn) settingsBtn.click();
    }, 100);
}

function deleteProject(projectId) {
    if (confirm('确定要删除这个项目吗？此操作不可恢复。')) {
        const index = PROJECTS.findIndex(p => p.id === projectId);
        if (index > -1) {
            PROJECTS.splice(index, 1);
            showToast('项目已删除', 'success');
            renderProjects();
        }
    }
}

function inviteMember(projectId) {
    const email = prompt('请输入要邀请的成员邮箱：');
    if (email) {
        showToast(`已向 ${email} 发送邀请`, 'success');
    }
}

function removeMember(projectId, memberName) {
    if (confirm(`确定要移除成员 ${memberName} 吗？`)) {
        showToast(`已移除成员 ${memberName}`, 'success');
    }
}

function addProjectsStyles() {
    if (document.getElementById('projectsStyles')) return;
    
    const style = document.createElement('style');
    style.id = 'projectsStyles';
    style.textContent = `
        .projects-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
            gap: 20px;
        }
        
        .project-card {
            background: var(--bg-base);
            border: 1px solid var(--border-default);
            border-radius: var(--radius-lg);
            overflow: hidden;
            cursor: pointer;
            transition: all var(--transition-fast);
        }
        
        .project-card:hover {
            border-color: var(--primary);
            transform: translateY(-2px);
            box-shadow: var(--shadow-glow);
        }
        
        .project-card.new-project {
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 250px;
            border-style: dashed;
        }
        
        .new-project-content {
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 12px;
            color: var(--text-tertiary);
        }
        
        .new-project-content i {
            font-size: 48px;
        }
        
        .project-card-header {
            display: flex;
            gap: 16px;
            padding: 20px;
            border-bottom: 1px solid var(--border-default);
        }
        
        .project-icon {
            width: 48px;
            height: 48px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: rgba(54,173,239,0.15);
            border-radius: var(--radius-md);
            color: var(--primary);
            font-size: 20px;
        }
        
        .project-info h3 {
            font-size: 18px;
            margin-bottom: 4px;
        }
        
        .project-info p {
            font-size: 13px;
            color: var(--text-secondary);
        }
        
        .project-card-body {
            padding: 20px;
        }
        
        .project-stats {
            display: flex;
            gap: 20px;
            margin-bottom: 16px;
        }
        
        .project-stat {
            display: flex;
            align-items: center;
            gap: 8px;
            font-size: 13px;
            color: var(--text-secondary);
        }
        
        .project-repo {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 10px 12px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
            font-size: 13px;
            margin-bottom: 16px;
        }
        
        .project-repo a {
            color: var(--primary);
        }
        
        .project-members {
            display: flex;
            align-items: center;
        }
        
        .member-avatar {
            width: 32px;
            height: 32px;
            border-radius: 50%;
            border: 2px solid var(--bg-base);
            margin-left: -8px;
        }
        
        .member-avatar:first-child {
            margin-left: 0;
        }
        
        .member-more {
            width: 32px;
            height: 32px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: var(--bg-elevated);
            border-radius: 50%;
            border: 2px solid var(--bg-base);
            margin-left: -8px;
            font-size: 11px;
            color: var(--text-tertiary);
        }
        
        .project-card-footer {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 12px 20px;
            background: var(--bg-elevated);
        }
        
        .project-date {
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .project-actions {
            display: flex;
            gap: 4px;
        }
        
        .members-list {
            display: flex;
            flex-direction: column;
            gap: 12px;
            margin-bottom: 16px;
        }
        
        .member-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .member-avatar-lg {
            width: 40px;
            height: 40px;
            border-radius: 50%;
        }
        
        .member-info {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        
        .member-name {
            font-weight: 500;
        }
        
        .member-role {
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .danger-zone {
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid var(--border-default);
        }
        
        .danger-zone h4 {
            color: var(--danger);
            margin-bottom: 12px;
        }
        
        .project-detail-info {
            padding: 20px 0;
        }
        
        .project-repo-link {
            display: flex;
            align-items: center;
            gap: 8px;
            margin: 16px 0;
            color: var(--text-secondary);
        }
        
        .project-repo-link a {
            color: var(--primary);
        }
        
        .project-meta {
            display: flex;
            gap: 20px;
            font-size: 13px;
            color: var(--text-tertiary);
        }
        
        .project-meta span {
            display: flex;
            align-items: center;
            gap: 6px;
        }
    `;
    document.head.appendChild(style);
}

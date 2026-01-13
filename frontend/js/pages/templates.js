// ===== 环境模板页面 =====

function renderTemplates() {
    const page = document.getElementById('templatesPage');
    
    page.innerHTML = `
        <div class="page-header">
            <div class="page-header-left">
                <h2>环境模板</h2>
                <p>选择预配置的开发环境模板，快速开始开发</p>
            </div>
            <div class="page-header-right">
                <div class="search-filter">
                    <input type="text" class="form-input" placeholder="搜索模板..." id="templateSearchInput" oninput="filterTemplatesPage()">
                </div>
            </div>
        </div>
        
        <!-- 分类标签 -->
        <div class="template-tabs">
            <button class="template-tab active" data-category="all" onclick="switchTemplateCategory('all')">
                <i class="fas fa-th-large"></i> 全部模板
                <span class="count">${TEMPLATES.length}</span>
            </button>
            <button class="template-tab" data-category="frontend" onclick="switchTemplateCategory('frontend')">
                <i class="fas fa-desktop"></i> 前端
                <span class="count">${TEMPLATES.filter(t => t.category === 'frontend').length}</span>
            </button>
            <button class="template-tab" data-category="backend" onclick="switchTemplateCategory('backend')">
                <i class="fas fa-server"></i> 后端
                <span class="count">${TEMPLATES.filter(t => t.category === 'backend').length}</span>
            </button>
            <button class="template-tab" data-category="database" onclick="switchTemplateCategory('database')">
                <i class="fas fa-database"></i> 数据库
                <span class="count">${TEMPLATES.filter(t => t.category === 'database').length}</span>
            </button>
            <button class="template-tab" data-category="mobile" onclick="switchTemplateCategory('mobile')">
                <i class="fas fa-mobile-alt"></i> 移动端
                <span class="count">${TEMPLATES.filter(t => t.category === 'mobile').length}</span>
            </button>
            <button class="template-tab" data-category="other" onclick="switchTemplateCategory('other')">
                <i class="fas fa-code"></i> 其他
                <span class="count">${TEMPLATES.filter(t => t.category === 'other').length}</span>
            </button>
        </div>
        
        <!-- 模板网格 -->
        <div class="templates-grid" id="templatesGrid">
            ${renderTemplateCards(TEMPLATES)}
        </div>
    `;
    
    addTemplatesStyles();
}

function renderTemplateCards(templates) {
    return templates.map(template => `
        <div class="template-card-large" onclick="createEnvFromTemplate('${template.id}')">
            <div class="template-card-icon">${template.icon}</div>
            <div class="template-card-content">
                <h3>${template.name}</h3>
                <span class="template-version">${template.version}</span>
                <p class="template-desc">${getTemplateDescription(template.id)}</p>
                <div class="template-tags">
                    <span class="tag">${getCategoryName(template.category)}</span>
                </div>
            </div>
            <div class="template-card-action">
                <button class="btn btn-primary">
                    <i class="fas fa-plus"></i> 使用此模板
                </button>
            </div>
        </div>
    `).join('');
}

function getTemplateDescription(templateId) {
    const descriptions = {
        'nodejs': '基于 Node.js 的服务端运行环境，适合构建高性能 Web 应用',
        'react': 'Facebook 开发的前端框架，用于构建用户界面',
        'vue': '渐进式 JavaScript 框架，易学易用',
        'angular': 'Google 开发的企业级前端框架',
        'nextjs': 'React 全栈框架，支持 SSR 和静态生成',
        'nuxtjs': 'Vue.js 全栈框架，支持 SSR 和静态生成',
        'svelte': '编译型前端框架，无虚拟 DOM',
        'express': '简洁灵活的 Node.js Web 框架',
        'springboot': 'Java 企业级开发框架，快速构建生产级应用',
        'django': 'Python 全功能 Web 框架，适合快速开发',
        'flask': 'Python 轻量级 Web 框架，灵活可扩展',
        'fastapi': 'Python 高性能 API 框架，支持异步',
        'go-gin': 'Go 语言高性能 Web 框架',
        'rust-rocket': 'Rust 语言 Web 框架，安全高效',
        'mysql': '流行的关系型数据库管理系统',
        'postgresql': '功能强大的开源关系型数据库',
        'mongodb': '文档型 NoSQL 数据库',
        'redis': '高性能内存数据库，支持多种数据结构',
        'react-native': '使用 React 构建原生移动应用',
        'flutter': 'Google 跨平台移动应用开发框架',
        'python': 'Python 通用开发环境',
        'java': 'Java 开发环境，支持多种框架',
        'cpp': 'C++ 开发环境，适合系统级编程',
        'csharp': '.NET 开发环境，支持跨平台',
        'php': 'PHP Web 开发环境',
        'ruby': 'Ruby 开发环境，适合 Web 开发',
        'kotlin': 'Kotlin 开发环境，Android 首选语言',
        'swift': 'Swift 开发环境，iOS/macOS 开发',
        'scala': 'Scala 开发环境，函数式编程',
        'clojure': 'Clojure 开发环境，Lisp 方言'
    };
    return descriptions[templateId] || '预配置的开发环境模板';
}

function getCategoryName(category) {
    const names = {
        'frontend': '前端',
        'backend': '后端',
        'database': '数据库',
        'mobile': '移动端',
        'other': '其他'
    };
    return names[category] || category;
}

function switchTemplateCategory(category) {
    // 更新标签状态
    document.querySelectorAll('.template-tab').forEach(tab => {
        tab.classList.toggle('active', tab.dataset.category === category);
    });
    
    // 过滤模板
    const filtered = category === 'all' 
        ? TEMPLATES 
        : TEMPLATES.filter(t => t.category === category);
    
    document.getElementById('templatesGrid').innerHTML = renderTemplateCards(filtered);
}

function filterTemplatesPage() {
    const searchTerm = document.getElementById('templateSearchInput').value.toLowerCase();
    const activeCategory = document.querySelector('.template-tab.active').dataset.category;
    
    let filtered = activeCategory === 'all' 
        ? TEMPLATES 
        : TEMPLATES.filter(t => t.category === activeCategory);
    
    if (searchTerm) {
        filtered = filtered.filter(t => 
            t.name.toLowerCase().includes(searchTerm) ||
            getTemplateDescription(t.id).toLowerCase().includes(searchTerm)
        );
    }
    
    document.getElementById('templatesGrid').innerHTML = renderTemplateCards(filtered);
}

function createEnvFromTemplate(templateId) {
    selectedTemplate = templateId;
    openModal('createEnvModal');
    
    // 跳到第二步
    setTimeout(() => {
        currentStep = 2;
        updateStepIndicator();
        showStepPanel(2);
        updateStepButtons();
        
        // 标记选中的模板
        document.querySelectorAll('.template-card').forEach(card => {
            card.classList.toggle('selected', card.dataset.id === templateId);
        });
    }, 100);
}

function addTemplatesStyles() {
    if (document.getElementById('templatesStyles')) return;
    
    const style = document.createElement('style');
    style.id = 'templatesStyles';
    style.textContent = `
        .template-tabs {
            display: flex;
            gap: 8px;
            margin-bottom: 24px;
            padding-bottom: 16px;
            border-bottom: 1px solid var(--border-default);
            overflow-x: auto;
        }
        
        .template-tab {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 10px 16px;
            background: var(--bg-elevated);
            border: 1px solid var(--border-default);
            border-radius: 20px;
            color: var(--text-secondary);
            font-size: 14px;
            white-space: nowrap;
            transition: all var(--transition-fast);
        }
        
        .template-tab:hover {
            border-color: var(--primary);
            color: var(--text-primary);
        }
        
        .template-tab.active {
            background: linear-gradient(135deg, var(--primary), var(--primary-dark));
            border-color: transparent;
            color: white;
        }
        
        .template-tab .count {
            padding: 2px 8px;
            background: rgba(255, 255, 255, 0.2);
            border-radius: 10px;
            font-size: 12px;
        }
        
        .template-tab.active .count {
            background: rgba(255, 255, 255, 0.3);
        }
        
        .templates-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
            gap: 20px;
        }
        
        .template-card-large {
            display: flex;
            flex-direction: column;
            background: var(--bg-base);
            border: 1px solid var(--border-default);
            border-radius: var(--radius-lg);
            overflow: hidden;
            cursor: pointer;
            transition: all var(--transition-fast);
        }
        
        .template-card-large:hover {
            border-color: var(--primary);
            transform: translateY(-4px);
            box-shadow: var(--shadow-glow);
        }
        
        .template-card-icon {
            display: flex;
            align-items: center;
            justify-content: center;
            height: 100px;
            background: var(--bg-elevated);
            font-size: 48px;
        }
        
        .template-card-content {
            flex: 1;
            padding: 20px;
        }
        
        .template-card-content h3 {
            font-size: 18px;
            margin-bottom: 4px;
        }
        
        .template-version {
            display: inline-block;
            padding: 2px 8px;
            background: var(--bg-elevated);
            border-radius: 4px;
            font-size: 12px;
            color: var(--text-tertiary);
            margin-bottom: 12px;
        }
        
        .template-desc {
            font-size: 14px;
            color: var(--text-secondary);
            line-height: 1.5;
            margin-bottom: 12px;
        }
        
        .template-tags {
            display: flex;
            gap: 8px;
        }
        
        .tag {
            padding: 4px 10px;
            background: rgba(54,173,239,0.1);
            border-radius: 4px;
            font-size: 12px;
            color: var(--primary);
        }
        
        .template-card-action {
            padding: 16px 20px;
            border-top: 1px solid var(--border-default);
        }
        
        .template-card-action .btn {
            width: 100%;
        }
        
        @media (max-width: 768px) {
            .templates-grid {
                grid-template-columns: 1fr;
            }
            
            .template-tabs {
                flex-wrap: nowrap;
                overflow-x: auto;
                -webkit-overflow-scrolling: touch;
            }
        }
    `;
    document.head.appendChild(style);
}

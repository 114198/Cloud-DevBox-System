// ===== 假数据 =====

// 模板数据
const TEMPLATES = [
    { id: 'nodejs', name: 'Node.js', icon: '🟢', category: 'backend', version: '18.x LTS' },
    { id: 'react', name: 'React', icon: '⚛️', category: 'frontend', version: '18.2' },
    { id: 'vue', name: 'Vue.js', icon: '💚', category: 'frontend', version: '3.3' },
    { id: 'angular', name: 'Angular', icon: '🅰️', category: 'frontend', version: '16.x' },
    { id: 'nextjs', name: 'Next.js', icon: '▲', category: 'frontend', version: '14.x' },
    { id: 'nuxtjs', name: 'Nuxt.js', icon: '💚', category: 'frontend', version: '3.x' },
    { id: 'svelte', name: 'Svelte', icon: '🔥', category: 'frontend', version: '4.x' },
    { id: 'express', name: 'Express.js', icon: '🚂', category: 'backend', version: '4.18' },
    { id: 'springboot', name: 'Spring Boot', icon: '🍃', category: 'backend', version: '3.1' },
    { id: 'django', name: 'Django', icon: '🐍', category: 'backend', version: '4.2' },
    { id: 'flask', name: 'Flask', icon: '🌶️', category: 'backend', version: '2.3' },
    { id: 'fastapi', name: 'FastAPI', icon: '⚡', category: 'backend', version: '0.100' },
    { id: 'go-gin', name: 'Go Gin', icon: '🐹', category: 'backend', version: '1.9' },
    { id: 'rust-rocket', name: 'Rust Rocket', icon: '🦀', category: 'backend', version: '0.5' },
    { id: 'mysql', name: 'MySQL', icon: '🐬', category: 'database', version: '8.0' },
    { id: 'postgresql', name: 'PostgreSQL', icon: '🐘', category: 'database', version: '15' },
    { id: 'mongodb', name: 'MongoDB', icon: '🍃', category: 'database', version: '6.0' },
    { id: 'redis', name: 'Redis', icon: '🔴', category: 'database', version: '7.0' },
    { id: 'react-native', name: 'React Native', icon: '📱', category: 'mobile', version: '0.72' },
    { id: 'flutter', name: 'Flutter', icon: '🦋', category: 'mobile', version: '3.10' },
    { id: 'python', name: 'Python', icon: '🐍', category: 'other', version: '3.11' },
    { id: 'java', name: 'Java', icon: '☕', category: 'other', version: '17 LTS' },
    { id: 'cpp', name: 'C++', icon: '⚙️', category: 'other', version: '20' },
    { id: 'csharp', name: 'C#', icon: '🎯', category: 'other', version: '.NET 7' },
    { id: 'php', name: 'PHP', icon: '🐘', category: 'other', version: '8.2' },
    { id: 'ruby', name: 'Ruby', icon: '💎', category: 'other', version: '3.2' },
    { id: 'kotlin', name: 'Kotlin', icon: '🎯', category: 'other', version: '1.9' },
    { id: 'swift', name: 'Swift', icon: '🍎', category: 'other', version: '5.9' },
    { id: 'scala', name: 'Scala', icon: '🔴', category: 'other', version: '3.3' },
    { id: 'clojure', name: 'Clojure', icon: '🔵', category: 'other', version: '1.11' }
];

// 环境数据
const ENVIRONMENTS = [
    {
        id: 'env-001',
        name: 'my-react-app',
        template: 'react',
        templateName: 'React',
        templateIcon: '⚛️',
        status: 'running',
        cpu: '2',
        memory: '4',
        storage: '20',
        domain: 'my-react-app.devbox.io',
        sshHost: 'ssh.devbox.io',
        sshPort: 22001,
        createdAt: '2026-01-05 10:30:00',
        lastAccessed: '2026-01-09 09:15:00',
        uptime: '4天 2小时',
        cpuUsage: 35,
        memoryUsage: 62,
        storageUsage: 45,
        gitRepo: 'https://github.com/user/my-react-app.git',
        collaborators: ['张三', '李四']
    },
    {
        id: 'env-002',
        name: 'backend-api',
        template: 'nodejs',
        templateName: 'Node.js',
        templateIcon: '🟢',
        status: 'running',
        cpu: '4',
        memory: '8',
        storage: '50',
        domain: 'backend-api.devbox.io',
        sshHost: 'ssh.devbox.io',
        sshPort: 22002,
        createdAt: '2026-01-03 14:20:00',
        lastAccessed: '2026-01-09 10:00:00',
        uptime: '6天 5小时',
        cpuUsage: 58,
        memoryUsage: 75,
        storageUsage: 32,
        gitRepo: 'https://github.com/user/backend-api.git',
        collaborators: ['张三']
    },
    {
        id: 'env-003',
        name: 'data-service',
        template: 'python',
        templateName: 'Python',
        templateIcon: '🐍',
        status: 'stopped',
        cpu: '2',
        memory: '4',
        storage: '30',
        domain: 'data-service.devbox.io',
        sshHost: 'ssh.devbox.io',
        sshPort: 22003,
        createdAt: '2026-01-01 09:00:00',
        lastAccessed: '2026-01-07 18:30:00',
        uptime: '-',
        cpuUsage: 0,
        memoryUsage: 0,
        storageUsage: 68,
        gitRepo: '',
        collaborators: []
    },
    {
        id: 'env-004',
        name: 'mobile-app',
        template: 'flutter',
        templateName: 'Flutter',
        templateIcon: '🦋',
        status: 'creating',
        cpu: '2',
        memory: '4',
        storage: '20',
        domain: '',
        sshHost: '',
        sshPort: 0,
        createdAt: '2026-01-09 10:25:00',
        lastAccessed: '-',
        uptime: '-',
        cpuUsage: 0,
        memoryUsage: 0,
        storageUsage: 5,
        gitRepo: '',
        collaborators: []
    },
    {
        id: 'env-005',
        name: 'legacy-project',
        template: 'java',
        templateName: 'Java',
        templateIcon: '☕',
        status: 'failed',
        cpu: '4',
        memory: '8',
        storage: '100',
        domain: '',
        sshHost: '',
        sshPort: 0,
        createdAt: '2026-01-08 16:00:00',
        lastAccessed: '-',
        uptime: '-',
        cpuUsage: 0,
        memoryUsage: 0,
        storageUsage: 0,
        gitRepo: 'https://github.com/user/legacy-project.git',
        collaborators: [],
        errorMessage: '资源配额不足，请升级套餐或释放其他环境'
    }
];

// 项目数据
const PROJECTS = [
    {
        id: 'proj-001',
        name: '电商平台',
        description: '全栈电商解决方案，包含前端商城和后台管理系统',
        environments: ['env-001', 'env-002'],
        members: [
            { name: '张三', role: 'owner', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan' },
            { name: '李四', role: 'editor', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=lisi' },
            { name: '王五', role: 'viewer', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=wangwu' }
        ],
        gitRepo: 'https://github.com/team/ecommerce-platform.git',
        createdAt: '2026-01-01',
        updatedAt: '2026-01-09'
    },
    {
        id: 'proj-002',
        name: '数据分析平台',
        description: '企业级数据分析和可视化平台',
        environments: ['env-003'],
        members: [
            { name: '张三', role: 'owner', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan' }
        ],
        gitRepo: 'https://github.com/team/data-analytics.git',
        createdAt: '2025-12-15',
        updatedAt: '2026-01-07'
    },
    {
        id: 'proj-003',
        name: '移动应用',
        description: '跨平台移动应用开发项目',
        environments: ['env-004'],
        members: [
            { name: '张三', role: 'owner', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan' },
            { name: '赵六', role: 'editor', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=zhaoliu' }
        ],
        gitRepo: 'https://github.com/team/mobile-app.git',
        createdAt: '2026-01-09',
        updatedAt: '2026-01-09'
    }
];

// 部署记录
const DEPLOYMENTS = [
    {
        id: 'deploy-001',
        envId: 'env-001',
        envName: 'my-react-app',
        version: 'v1.2.3',
        status: 'success',
        startTime: '2026-01-09 09:00:00',
        endTime: '2026-01-09 09:03:25',
        duration: '3分25秒',
        triggeredBy: '张三',
        commitId: 'a1b2c3d',
        commitMessage: 'feat: 添加用户登录功能'
    },
    {
        id: 'deploy-002',
        envId: 'env-002',
        envName: 'backend-api',
        version: 'v2.0.1',
        status: 'success',
        startTime: '2026-01-08 18:30:00',
        endTime: '2026-01-08 18:35:12',
        duration: '5分12秒',
        triggeredBy: '张三',
        commitId: 'e4f5g6h',
        commitMessage: 'fix: 修复API响应超时问题'
    },
    {
        id: 'deploy-003',
        envId: 'env-001',
        envName: 'my-react-app',
        version: 'v1.2.2',
        status: 'failed',
        startTime: '2026-01-08 14:00:00',
        endTime: '2026-01-08 14:02:30',
        duration: '2分30秒',
        triggeredBy: '李四',
        commitId: 'i7j8k9l',
        commitMessage: 'feat: 新增购物车功能',
        errorMessage: '构建失败：依赖包版本冲突'
    },
    {
        id: 'deploy-004',
        envId: 'env-002',
        envName: 'backend-api',
        version: 'v2.0.0',
        status: 'success',
        startTime: '2026-01-07 10:00:00',
        endTime: '2026-01-07 10:04:45',
        duration: '4分45秒',
        triggeredBy: '张三',
        commitId: 'm0n1o2p',
        commitMessage: 'release: 发布2.0版本'
    }
];

// 协作会话
const COLLABORATION_SESSIONS = [
    {
        id: 'collab-001',
        envId: 'env-001',
        envName: 'my-react-app',
        participants: [
            { name: '张三', status: 'active', cursor: { file: 'src/App.tsx', line: 42 } },
            { name: '李四', status: 'active', cursor: { file: 'src/components/Header.tsx', line: 15 } }
        ],
        startTime: '2026-01-09 09:00:00',
        activeFiles: ['src/App.tsx', 'src/components/Header.tsx', 'src/styles/main.css']
    },
    {
        id: 'collab-002',
        envId: 'env-002',
        envName: 'backend-api',
        participants: [
            { name: '张三', status: 'idle', cursor: { file: 'src/routes/api.js', line: 88 } }
        ],
        startTime: '2026-01-09 08:30:00',
        activeFiles: ['src/routes/api.js', 'src/models/user.js']
    }
];

// 监控数据
const MONITORING_DATA = {
    systemMetrics: {
        totalEnvironments: 5,
        runningEnvironments: 2,
        totalCPU: 14,
        usedCPU: 6,
        totalMemory: 28,
        usedMemory: 12,
        totalStorage: 220,
        usedStorage: 85
    },
    alerts: [
        { id: 'alert-001', type: 'warning', message: 'env-002 内存使用率超过75%', time: '2026-01-09 09:30:00' },
        { id: 'alert-002', type: 'info', message: 'env-003 已停止超过48小时', time: '2026-01-09 08:00:00' },
        { id: 'alert-003', type: 'error', message: 'env-005 创建失败', time: '2026-01-08 16:05:00' }
    ],
    cpuHistory: [35, 42, 38, 45, 52, 48, 55, 50, 45, 42, 38, 35],
    memoryHistory: [60, 62, 65, 68, 70, 72, 75, 73, 70, 68, 65, 62],
    networkHistory: [120, 150, 180, 200, 220, 250, 280, 260, 240, 220, 200, 180]
};

// 账单数据
const BILLING_DATA = {
    currentMonth: {
        total: 156.80,
        cpu: 72.00,
        memory: 48.00,
        storage: 36.80,
        freeQuota: 20,
        usedFreeQuota: 20
    },
    history: [
        { month: '2025-12', total: 142.50, paid: true },
        { month: '2025-11', total: 128.30, paid: true },
        { month: '2025-10', total: 98.60, paid: true },
        { month: '2025-09', total: 85.20, paid: true }
    ],
    usageDetails: [
        { date: '2026-01-09', cpu: 8.00, memory: 5.33, storage: 4.10, total: 17.43 },
        { date: '2026-01-08', cpu: 7.50, memory: 5.00, storage: 4.10, total: 16.60 },
        { date: '2026-01-07', cpu: 7.00, memory: 4.67, storage: 4.00, total: 15.67 },
        { date: '2026-01-06', cpu: 6.50, memory: 4.33, storage: 3.90, total: 14.73 },
        { date: '2026-01-05', cpu: 6.00, memory: 4.00, storage: 3.80, total: 13.80 }
    ]
};

// 用户设置
const USER_SETTINGS = {
    profile: {
        username: '张三',
        email: 'zhangsan@example.com',
        displayName: '张三',
        avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=user',
        role: 'developer',
        organization: '示例科技有限公司'
    },
    preferences: {
        language: 'zh-CN',
        timezone: 'Asia/Shanghai',
        theme: 'dark',
        notifications: {
            email: true,
            browser: true,
            deploySuccess: true,
            deployFailed: true,
            resourceAlert: true
        }
    },
    security: {
        mfaEnabled: false,
        lastPasswordChange: '2025-12-01',
        activeSessions: 2
    },
    sshKeys: [
        { id: 'key-001', name: 'MacBook Pro', fingerprint: 'SHA256:abc123...', createdAt: '2025-11-15' },
        { id: 'key-002', name: 'Windows Desktop', fingerprint: 'SHA256:def456...', createdAt: '2025-12-20' }
    ],
    apiKeys: [
        { id: 'api-001', name: '开发测试', prefix: 'devbox_sk_test_', createdAt: '2025-12-01', lastUsed: '2026-01-08' }
    ]
};

// 通知数据
const NOTIFICATIONS = [
    { id: 'notif-001', type: 'success', title: '部署成功', message: 'my-react-app v1.2.3 部署完成', time: '10分钟前', read: false },
    { id: 'notif-002', type: 'warning', title: '资源告警', message: 'backend-api 内存使用率超过75%', time: '30分钟前', read: false },
    { id: 'notif-003', type: 'info', title: '协作邀请', message: '李四邀请您加入项目"电商平台"', time: '1小时前', read: true }
];

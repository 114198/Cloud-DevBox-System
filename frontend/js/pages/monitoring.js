// ===== 监控中心页面 =====

function renderMonitoring() {
    const page = document.getElementById('monitoringPage');
    const metrics = MONITORING_DATA.systemMetrics;
    
    page.innerHTML = `
        <div class="page-header">
            <div class="page-header-left">
                <h2>监控中心</h2>
                <p>系统资源监控和告警管理</p>
            </div>
            <div class="page-header-right">
                <select class="form-input" style="width: 150px;">
                    <option>最近1小时</option>
                    <option>最近6小时</option>
                    <option>最近24小时</option>
                    <option>最近7天</option>
                </select>
                <button class="btn btn-outline" onclick="refreshMonitoring()">
                    <i class="fas fa-sync-alt"></i> 刷新
                </button>
            </div>
        </div>
        
        <!-- 资源概览 -->
        <div class="monitoring-overview">
            <div class="resource-card">
                <div class="resource-header">
                    <span class="resource-title">CPU 使用率</span>
                    <span class="resource-value">${Math.round(metrics.usedCPU / metrics.totalCPU * 100)}%</span>
                </div>
                <div class="resource-chart">
                    <div class="mini-chart" id="cpuChart"></div>
                </div>
                <div class="resource-detail">
                    <span>已使用: ${metrics.usedCPU} 核</span>
                    <span>总计: ${metrics.totalCPU} 核</span>
                </div>
            </div>
            
            <div class="resource-card">
                <div class="resource-header">
                    <span class="resource-title">内存使用率</span>
                    <span class="resource-value">${Math.round(metrics.usedMemory / metrics.totalMemory * 100)}%</span>
                </div>
                <div class="resource-chart">
                    <div class="mini-chart" id="memoryChart"></div>
                </div>
                <div class="resource-detail">
                    <span>已使用: ${metrics.usedMemory} GB</span>
                    <span>总计: ${metrics.totalMemory} GB</span>
                </div>
            </div>
            
            <div class="resource-card">
                <div class="resource-header">
                    <span class="resource-title">存储使用率</span>
                    <span class="resource-value">${Math.round(metrics.usedStorage / metrics.totalStorage * 100)}%</span>
                </div>
                <div class="resource-chart">
                    <div class="mini-chart" id="storageChart"></div>
                </div>
                <div class="resource-detail">
                    <span>已使用: ${metrics.usedStorage} GB</span>
                    <span>总计: ${metrics.totalStorage} GB</span>
                </div>
            </div>
            
            <div class="resource-card">
                <div class="resource-header">
                    <span class="resource-title">环境状态</span>
                    <span class="resource-value">${metrics.runningEnvironments}/${metrics.totalEnvironments}</span>
                </div>
                <div class="env-status-chart">
                    <div class="status-bar">
                        <div class="status-segment running" style="width: ${metrics.runningEnvironments / metrics.totalEnvironments * 100}%"></div>
                        <div class="status-segment stopped" style="width: ${(metrics.totalEnvironments - metrics.runningEnvironments) / metrics.totalEnvironments * 100}%"></div>
                    </div>
                </div>
                <div class="resource-detail">
                    <span class="running">运行中: ${metrics.runningEnvironments}</span>
                    <span class="stopped">已停止: ${metrics.totalEnvironments - metrics.runningEnvironments}</span>
                </div>
            </div>
        </div>
        
        <!-- 图表区域 -->
        <div class="charts-grid">
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">CPU 使用趋势</h3>
                </div>
                <div class="card-body">
                    <div class="chart-container">
                        <canvas id="cpuTrendChart"></canvas>
                    </div>
                </div>
            </div>
            
            <div class="card">
                <div class="card-header">
                    <h3 class="card-title">内存使用趋势</h3>
                </div>
                <div class="card-body">
                    <div class="chart-container">
                        <canvas id="memoryTrendChart"></canvas>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 告警列表 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">
                    <i class="fas fa-bell"></i>
                    系统告警
                    <span class="alert-count">${MONITORING_DATA.alerts.length}</span>
                </h3>
                <button class="btn btn-outline btn-sm" onclick="clearAllAlerts()">
                    全部已读
                </button>
            </div>
            <div class="card-body">
                <div class="alerts-list">
                    ${MONITORING_DATA.alerts.map(alert => `
                        <div class="alert-item ${alert.type}">
                            <div class="alert-icon">
                                <i class="fas ${alert.type === 'error' ? 'fa-times-circle' : alert.type === 'warning' ? 'fa-exclamation-triangle' : 'fa-info-circle'}"></i>
                            </div>
                            <div class="alert-content">
                                <span class="alert-message">${alert.message}</span>
                                <span class="alert-time">${formatRelativeTime(alert.time)}</span>
                            </div>
                            <button class="btn-icon" onclick="dismissAlert('${alert.id}')" title="忽略">
                                <i class="fas fa-times"></i>
                            </button>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
        
        <!-- 环境资源详情 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">环境资源详情</h3>
            </div>
            <div class="card-body">
                <div class="table-container">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>环境名称</th>
                                <th>状态</th>
                                <th>CPU</th>
                                <th>内存</th>
                                <th>存储</th>
                                <th>运行时长</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${ENVIRONMENTS.map(env => `
                                <tr>
                                    <td>
                                        <div class="env-name-cell">
                                            <span class="env-icon">${env.templateIcon}</span>
                                            <span>${env.name}</span>
                                        </div>
                                    </td>
                                    <td>
                                        <span class="status-badge ${env.status}">
                                            <span class="status-dot"></span>
                                            ${getStatusText(env.status)}
                                        </span>
                                    </td>
                                    <td>
                                        <div class="usage-cell">
                                            <div class="progress-bar small">
                                                <div class="progress-fill ${env.cpuUsage > 80 ? 'danger' : env.cpuUsage > 60 ? 'warning' : ''}" 
                                                     style="width: ${env.cpuUsage}%"></div>
                                            </div>
                                            <span>${env.cpuUsage}%</span>
                                        </div>
                                    </td>
                                    <td>
                                        <div class="usage-cell">
                                            <div class="progress-bar small">
                                                <div class="progress-fill ${env.memoryUsage > 80 ? 'danger' : env.memoryUsage > 60 ? 'warning' : ''}" 
                                                     style="width: ${env.memoryUsage}%"></div>
                                            </div>
                                            <span>${env.memoryUsage}%</span>
                                        </div>
                                    </td>
                                    <td>
                                        <div class="usage-cell">
                                            <div class="progress-bar small">
                                                <div class="progress-fill ${env.storageUsage > 80 ? 'danger' : env.storageUsage > 60 ? 'warning' : ''}" 
                                                     style="width: ${env.storageUsage}%"></div>
                                            </div>
                                            <span>${env.storageUsage}%</span>
                                        </div>
                                    </td>
                                    <td>${env.uptime}</td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    `;
    
    addMonitoringStyles();
    renderMiniCharts();
}

function renderMiniCharts() {
    // 简单的 SVG 迷你图表
    const cpuData = MONITORING_DATA.cpuHistory;
    const memoryData = MONITORING_DATA.memoryHistory;
    
    renderSparkline('cpuChart', cpuData, '#3b82f6');
    renderSparkline('memoryChart', memoryData, '#10b981');
}

function renderSparkline(containerId, data, color) {
    const container = document.getElementById(containerId);
    if (!container) return;
    
    const width = 150;
    const height = 40;
    const max = Math.max(...data);
    const min = Math.min(...data);
    const range = max - min || 1;
    
    const points = data.map((value, index) => {
        const x = (index / (data.length - 1)) * width;
        const y = height - ((value - min) / range) * height;
        return `${x},${y}`;
    }).join(' ');
    
    container.innerHTML = `
        <svg width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">
            <polyline
                fill="none"
                stroke="${color}"
                stroke-width="2"
                points="${points}"
            />
        </svg>
    `;
}

function refreshMonitoring() {
    showToast('正在刷新监控数据...', 'info');
    setTimeout(() => {
        renderMonitoring();
        showToast('监控数据已更新', 'success');
    }, 1000);
}

function clearAllAlerts() {
    MONITORING_DATA.alerts = [];
    renderMonitoring();
    showToast('所有告警已清除', 'success');
}

function dismissAlert(alertId) {
    const index = MONITORING_DATA.alerts.findIndex(a => a.id === alertId);
    if (index > -1) {
        MONITORING_DATA.alerts.splice(index, 1);
        renderMonitoring();
    }
}

function addMonitoringStyles() {
    if (document.getElementById('monitoringStyles')) return;
    
    const style = document.createElement('style');
    style.id = 'monitoringStyles';
    style.textContent = `
        .monitoring-overview {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 20px;
            margin-bottom: 24px;
        }
        
        .resource-card {
            background: var(--bg-base);
            border: 1px solid var(--border-default);
            border-radius: var(--radius-lg);
            padding: 20px;
        }
        
        .resource-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
        }
        
        .resource-title {
            font-size: 14px;
            color: var(--text-secondary);
        }
        
        .resource-value {
            font-size: 24px;
            font-weight: 700;
        }
        
        .resource-chart {
            margin-bottom: 16px;
        }
        
        .mini-chart {
            height: 40px;
        }
        
        .resource-detail {
            display: flex;
            justify-content: space-between;
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .resource-detail .running {
            color: var(--success);
        }
        
        .resource-detail .stopped {
            color: var(--text-tertiary);
        }
        
        .env-status-chart {
            margin-bottom: 16px;
        }
        
        .status-bar {
            display: flex;
            height: 8px;
            border-radius: 4px;
            overflow: hidden;
            background: var(--bg-elevated);
        }
        
        .status-segment.running {
            background: var(--success);
        }
        
        .status-segment.stopped {
            background: var(--text-tertiary);
        }
        
        .charts-grid {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 20px;
            margin-bottom: 24px;
        }
        
        .chart-container {
            height: 200px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .chart-container::after {
            content: '图表加载中...';
            color: var(--text-tertiary);
        }
        
        .alert-count {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            min-width: 20px;
            height: 20px;
            padding: 0 6px;
            background: var(--danger);
            border-radius: 10px;
            font-size: 11px;
            color: white;
            margin-left: 8px;
        }
        
        .alerts-list {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }
        
        .alert-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 16px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
            border-left: 4px solid;
        }
        
        .alert-item.error {
            border-left-color: var(--danger);
        }
        
        .alert-item.warning {
            border-left-color: var(--warning);
        }
        
        .alert-item.info {
            border-left-color: var(--info);
        }
        
        .alert-icon {
            font-size: 18px;
        }
        
        .alert-item.error .alert-icon { color: var(--danger); }
        .alert-item.warning .alert-icon { color: var(--warning); }
        .alert-item.info .alert-icon { color: var(--info); }
        
        .alert-content {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        
        .alert-message {
            font-size: 14px;
        }
        
        .alert-time {
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .env-name-cell {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .env-icon {
            font-size: 18px;
        }
        
        .usage-cell {
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .progress-bar.small {
            width: 60px;
            height: 6px;
        }
        
        @media (max-width: 1200px) {
            .monitoring-overview {
                grid-template-columns: repeat(2, 1fr);
            }
            
            .charts-grid {
                grid-template-columns: 1fr;
            }
        }
        
        @media (max-width: 768px) {
            .monitoring-overview {
                grid-template-columns: 1fr;
            }
        }
    `;
    document.head.appendChild(style);
}

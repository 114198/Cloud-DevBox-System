// ===== 费用账单页面 =====

function renderBilling() {
    const page = document.getElementById('billingPage');
    const billing = BILLING_DATA;
    
    page.innerHTML = `
        <div class="page-header">
            <div class="page-header-left">
                <h2>费用账单</h2>
                <p>查看资源使用和费用明细</p>
            </div>
            <div class="page-header-right">
                <button class="btn btn-outline" onclick="downloadInvoice()">
                    <i class="fas fa-download"></i> 下载账单
                </button>
                <button class="btn btn-primary" onclick="openRechargeModal()">
                    <i class="fas fa-plus"></i> 充值
                </button>
            </div>
        </div>
        
        <!-- 费用概览 -->
        <div class="billing-overview">
            <div class="billing-card main">
                <div class="billing-card-header">
                    <span class="billing-label">本月费用</span>
                    <span class="billing-period">2026年1月</span>
                </div>
                <div class="billing-amount">
                    <span class="currency">¥</span>
                    <span class="amount">${billing.currentMonth.total.toFixed(2)}</span>
                </div>
                <div class="billing-breakdown">
                    <div class="breakdown-item">
                        <span class="breakdown-label">CPU</span>
                        <span class="breakdown-value">¥${billing.currentMonth.cpu.toFixed(2)}</span>
                    </div>
                    <div class="breakdown-item">
                        <span class="breakdown-label">内存</span>
                        <span class="breakdown-value">¥${billing.currentMonth.memory.toFixed(2)}</span>
                    </div>
                    <div class="breakdown-item">
                        <span class="breakdown-label">存储</span>
                        <span class="breakdown-value">¥${billing.currentMonth.storage.toFixed(2)}</span>
                    </div>
                </div>
            </div>
            
            <div class="billing-card">
                <div class="billing-card-header">
                    <span class="billing-label">免费额度</span>
                </div>
                <div class="free-quota">
                    <div class="quota-progress">
                        <div class="progress-bar">
                            <div class="progress-fill" style="width: ${billing.currentMonth.usedFreeQuota / billing.currentMonth.freeQuota * 100}%"></div>
                        </div>
                        <span class="quota-text">${billing.currentMonth.usedFreeQuota}/${billing.currentMonth.freeQuota} 小时</span>
                    </div>
                    <p class="quota-note">每月免费使用 ${billing.currentMonth.freeQuota} 小时</p>
                </div>
            </div>
            
            <div class="billing-card">
                <div class="billing-card-header">
                    <span class="billing-label">账户余额</span>
                </div>
                <div class="balance-info">
                    <span class="balance-amount">¥ 500.00</span>
                    <button class="btn btn-sm btn-primary" onclick="openRechargeModal()">充值</button>
                </div>
                <p class="balance-note">预计可使用约 200 小时</p>
            </div>
        </div>
        
        <!-- 价格说明 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">价格说明</h3>
            </div>
            <div class="card-body">
                <div class="pricing-table">
                    <div class="pricing-item">
                        <div class="pricing-icon">
                            <i class="fas fa-microchip"></i>
                        </div>
                        <div class="pricing-info">
                            <span class="pricing-name">CPU</span>
                            <span class="pricing-price">¥0.10 / 核 / 小时</span>
                        </div>
                    </div>
                    <div class="pricing-item">
                        <div class="pricing-icon">
                            <i class="fas fa-memory"></i>
                        </div>
                        <div class="pricing-info">
                            <span class="pricing-name">内存</span>
                            <span class="pricing-price">¥0.05 / GB / 小时</span>
                        </div>
                    </div>
                    <div class="pricing-item">
                        <div class="pricing-icon">
                            <i class="fas fa-hdd"></i>
                        </div>
                        <div class="pricing-info">
                            <span class="pricing-name">存储</span>
                            <span class="pricing-price">¥0.01 / GB / 小时</span>
                        </div>
                    </div>
                    <div class="pricing-item">
                        <div class="pricing-icon">
                            <i class="fas fa-network-wired"></i>
                        </div>
                        <div class="pricing-info">
                            <span class="pricing-name">流量</span>
                            <span class="pricing-price">免费 (100GB/月)</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 使用明细 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">使用明细</h3>
                <select class="form-input" style="width: 150px;">
                    <option>本月</option>
                    <option>上月</option>
                    <option>最近3个月</option>
                </select>
            </div>
            <div class="card-body">
                <div class="table-container">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>日期</th>
                                <th>CPU 费用</th>
                                <th>内存费用</th>
                                <th>存储费用</th>
                                <th>合计</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${billing.usageDetails.map(detail => `
                                <tr>
                                    <td>${detail.date}</td>
                                    <td>¥${detail.cpu.toFixed(2)}</td>
                                    <td>¥${detail.memory.toFixed(2)}</td>
                                    <td>¥${detail.storage.toFixed(2)}</td>
                                    <td><strong>¥${detail.total.toFixed(2)}</strong></td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
        
        <!-- 历史账单 -->
        <div class="card">
            <div class="card-header">
                <h3 class="card-title">历史账单</h3>
            </div>
            <div class="card-body">
                <div class="history-bills">
                    ${billing.history.map(bill => `
                        <div class="bill-item">
                            <div class="bill-info">
                                <span class="bill-month">${bill.month}</span>
                                <span class="bill-amount">¥${bill.total.toFixed(2)}</span>
                            </div>
                            <div class="bill-status ${bill.paid ? 'paid' : 'unpaid'}">
                                ${bill.paid ? '已支付' : '待支付'}
                            </div>
                            <div class="bill-actions">
                                <button class="btn btn-sm btn-outline" onclick="viewBillDetail('${bill.month}')">
                                    查看详情
                                </button>
                                <button class="btn btn-sm btn-outline" onclick="downloadBill('${bill.month}')">
                                    <i class="fas fa-download"></i>
                                </button>
                            </div>
                        </div>
                    `).join('')}
                </div>
            </div>
        </div>
    `;
    
    addBillingStyles();
}

function openRechargeModal() {
    const modal = document.createElement('div');
    modal.className = 'modal active';
    modal.id = 'rechargeModal';
    modal.innerHTML = `
        <div class="modal-overlay" onclick="closeModal('rechargeModal')"></div>
        <div class="modal-content">
            <div class="modal-header">
                <h3>账户充值</h3>
                <button class="modal-close" onclick="closeModal('rechargeModal')">
                    <i class="fas fa-times"></i>
                </button>
            </div>
            <div class="modal-body">
                <div class="recharge-options">
                    <div class="recharge-option" onclick="selectRechargeAmount(100)">
                        <span class="recharge-amount">¥100</span>
                    </div>
                    <div class="recharge-option" onclick="selectRechargeAmount(200)">
                        <span class="recharge-amount">¥200</span>
                    </div>
                    <div class="recharge-option selected" onclick="selectRechargeAmount(500)">
                        <span class="recharge-amount">¥500</span>
                        <span class="recharge-bonus">送50元</span>
                    </div>
                    <div class="recharge-option" onclick="selectRechargeAmount(1000)">
                        <span class="recharge-amount">¥1000</span>
                        <span class="recharge-bonus">送150元</span>
                    </div>
                </div>
                
                <div class="form-group">
                    <label>自定义金额</label>
                    <input type="number" id="customAmount" class="form-input" placeholder="输入充值金额">
                </div>
                
                <div class="payment-methods">
                    <h4>支付方式</h4>
                    <div class="payment-options">
                        <label class="payment-option selected">
                            <input type="radio" name="payment" value="alipay" checked>
                            <i class="fab fa-alipay"></i>
                            <span>支付宝</span>
                        </label>
                        <label class="payment-option">
                            <input type="radio" name="payment" value="wechat">
                            <i class="fab fa-weixin"></i>
                            <span>微信支付</span>
                        </label>
                        <label class="payment-option">
                            <input type="radio" name="payment" value="card">
                            <i class="fas fa-credit-card"></i>
                            <span>银行卡</span>
                        </label>
                    </div>
                </div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-outline" onclick="closeModal('rechargeModal')">取消</button>
                <button class="btn btn-primary" onclick="processRecharge()">
                    确认充值 ¥<span id="rechargeTotal">500</span>
                </button>
            </div>
        </div>
    `;
    document.body.appendChild(modal);
}

function selectRechargeAmount(amount) {
    document.querySelectorAll('.recharge-option').forEach(opt => {
        opt.classList.remove('selected');
    });
    event.currentTarget.classList.add('selected');
    document.getElementById('rechargeTotal').textContent = amount;
    document.getElementById('customAmount').value = '';
}

function processRecharge() {
    showToast('正在处理支付...', 'info');
    setTimeout(() => {
        closeModal('rechargeModal');
        showToast('充值成功！', 'success');
    }, 2000);
}

function downloadInvoice() {
    showToast('正在生成账单...', 'info');
    setTimeout(() => {
        showToast('账单已下载', 'success');
    }, 1000);
}

function viewBillDetail(month) {
    showToast(`查看 ${month} 账单详情`, 'info');
}

function downloadBill(month) {
    showToast(`下载 ${month} 账单`, 'info');
}

function addBillingStyles() {
    if (document.getElementById('billingStyles')) return;
    
    const style = document.createElement('style');
    style.id = 'billingStyles';
    style.textContent = `
        .billing-overview {
            display: grid;
            grid-template-columns: 2fr 1fr 1fr;
            gap: 20px;
            margin-bottom: 24px;
        }
        
        .billing-card {
            background: var(--bg-base);
            border: 1px solid var(--border-default);
            border-radius: var(--radius-lg);
            padding: 24px;
        }
        
        .billing-card.main {
            background: linear-gradient(135deg, var(--primary), var(--secondary));
        }
        
        .billing-card-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
        }
        
        .billing-label {
            font-size: 14px;
            opacity: 0.9;
        }
        
        .billing-period {
            font-size: 12px;
            opacity: 0.7;
        }
        
        .billing-amount {
            margin-bottom: 20px;
        }
        
        .billing-amount .currency {
            font-size: 20px;
            vertical-align: top;
        }
        
        .billing-amount .amount {
            font-size: 42px;
            font-weight: 700;
        }
        
        .billing-breakdown {
            display: flex;
            gap: 20px;
            padding-top: 16px;
            border-top: 1px solid rgba(255, 255, 255, 0.2);
        }
        
        .breakdown-item {
            display: flex;
            flex-direction: column;
        }
        
        .breakdown-label {
            font-size: 12px;
            opacity: 0.7;
        }
        
        .breakdown-value {
            font-size: 16px;
            font-weight: 600;
        }
        
        .free-quota {
            margin-top: 16px;
        }
        
        .quota-progress {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-bottom: 8px;
        }
        
        .quota-progress .progress-bar {
            flex: 1;
        }
        
        .quota-text {
            font-size: 14px;
            font-weight: 500;
        }
        
        .quota-note {
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .balance-info {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin: 16px 0;
        }
        
        .balance-amount {
            font-size: 28px;
            font-weight: 700;
        }
        
        .balance-note {
            font-size: 12px;
            color: var(--text-tertiary);
        }
        
        .pricing-table {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 16px;
        }
        
        .pricing-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 16px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .pricing-icon {
            width: 40px;
            height: 40px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: rgba(54,173,239,0.15);
            border-radius: var(--radius-md);
            color: var(--primary);
        }
        
        .pricing-info {
            display: flex;
            flex-direction: column;
        }
        
        .pricing-name {
            font-weight: 500;
        }
        
        .pricing-price {
            font-size: 13px;
            color: var(--text-secondary);
        }
        
        .history-bills {
            display: flex;
            flex-direction: column;
            gap: 12px;
        }
        
        .bill-item {
            display: flex;
            align-items: center;
            gap: 16px;
            padding: 16px;
            background: var(--bg-elevated);
            border-radius: var(--radius-md);
        }
        
        .bill-info {
            flex: 1;
            display: flex;
            flex-direction: column;
        }
        
        .bill-month {
            font-weight: 500;
        }
        
        .bill-amount {
            font-size: 18px;
            font-weight: 700;
        }
        
        .bill-status {
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 12px;
        }
        
        .bill-status.paid {
            background: rgba(0,217,181,0.15);
            color: var(--success);
        }
        
        .bill-status.unpaid {
            background: rgba(255,176,32,0.15);
            color: var(--warning);
        }
        
        .bill-actions {
            display: flex;
            gap: 8px;
        }
        
        .recharge-options {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 12px;
            margin-bottom: 20px;
        }
        
        .recharge-option {
            display: flex;
            flex-direction: column;
            align-items: center;
            padding: 16px;
            background: var(--bg-elevated);
            border: 2px solid var(--border-default);
            border-radius: var(--radius-md);
            cursor: pointer;
            transition: all var(--transition-fast);
        }
        
        .recharge-option:hover {
            border-color: var(--primary);
        }
        
        .recharge-option.selected {
            border-color: var(--primary);
            background: rgba(54,173,239,0.1);
        }
        
        .recharge-amount {
            font-size: 20px;
            font-weight: 700;
        }
        
        .recharge-bonus {
            font-size: 12px;
            color: var(--success);
            margin-top: 4px;
        }
        
        .payment-methods h4 {
            margin-bottom: 12px;
            font-size: 14px;
            color: var(--text-secondary);
        }
        
        .payment-options {
            display: flex;
            gap: 12px;
        }
        
        .payment-option {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 8px;
            padding: 16px;
            background: var(--bg-elevated);
            border: 2px solid var(--border-default);
            border-radius: var(--radius-md);
            cursor: pointer;
            transition: all var(--transition-fast);
        }
        
        .payment-option:hover {
            border-color: var(--primary);
        }
        
        .payment-option.selected {
            border-color: var(--primary);
        }
        
        .payment-option input {
            display: none;
        }
        
        .payment-option i {
            font-size: 24px;
        }
        
        @media (max-width: 1024px) {
            .billing-overview {
                grid-template-columns: 1fr;
            }
            
            .pricing-table {
                grid-template-columns: repeat(2, 1fr);
            }
        }
        
        @media (max-width: 768px) {
            .pricing-table {
                grid-template-columns: 1fr;
            }
            
            .recharge-options {
                grid-template-columns: repeat(2, 1fr);
            }
        }
    `;
    document.head.appendChild(style);
}

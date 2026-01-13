import { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Modal,
  Form,
  Select,
  InputNumber,
  Switch,
  message,
  Empty,
  Tooltip,
  Badge,
  Popconfirm,
} from 'antd'
import {
  BellOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  WarningOutlined,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
} from '@ant-design/icons'
import {
  monitoringService,
  Alert,
  AlertRule,
  AlertSeverity,
  AlertStatus,
  ListAlertsRequest,
} from '@/services/monitoring'

interface AlertPanelProps {
  environmentId?: string
  showRules?: boolean
}

const severityConfig: Record<AlertSeverity, { color: string; icon: React.ReactNode; text: string }> = {
  critical: { color: 'red', icon: <ExclamationCircleOutlined />, text: '严重' },
  warning: { color: 'orange', icon: <WarningOutlined />, text: '警告' },
  info: { color: 'blue', icon: <BellOutlined />, text: '信息' },
}

const statusConfig: Record<AlertStatus, { color: string; text: string }> = {
  active: { color: 'red', text: '活跃' },
  resolved: { color: 'green', text: '已解决' },
  acknowledged: { color: 'blue', text: '已确认' },
}

const metricTypeOptions = [
  { value: 'cpu', label: 'CPU 使用率' },
  { value: 'memory', label: '内存使用率' },
  { value: 'storage', label: '存储使用率' },
  { value: 'network', label: '网络流量' },
]

export default function AlertPanel({ environmentId, showRules = true }: AlertPanelProps) {
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [rules, setRules] = useState<AlertRule[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isRuleModalOpen, setIsRuleModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null)
  const [form] = Form.useForm()

  useEffect(() => {
    loadData()
  }, [environmentId])

  const loadData = async () => {
    setIsLoading(true)
    try {
      const params: ListAlertsRequest = { page: 1, pageSize: 50 }
      if (environmentId) params.environmentId = environmentId

      const [alertsRes, rulesRes] = await Promise.all([
        monitoringService.listAlerts(params),
        showRules ? monitoringService.listAlertRules() : Promise.resolve([]),
      ])
      setAlerts(alertsRes.alerts || [])
      setRules(rulesRes || [])
    } catch (error) {
      console.error('Failed to load alerts:', error)
      // Use mock data for demo
      setAlerts(generateMockAlerts(environmentId))
      setRules(generateMockRules())
    } finally {
      setIsLoading(false)
    }
  }

  const handleAcknowledge = async (alertId: string) => {
    try {
      await monitoringService.acknowledgeAlert(alertId)
      message.success('告警已确认')
      loadData()
    } catch (error) {
      message.error('确认告警失败')
    }
  }

  const handleCreateRule = () => {
    setEditingRule(null)
    form.resetFields()
    form.setFieldsValue({
      type: 'storage',
      severity: 'warning',
      threshold: 90,
      duration: '5m',
      notifyMethods: ['web'],
      enabled: true,
    })
    setIsRuleModalOpen(true)
  }

  const handleEditRule = (rule: AlertRule) => {
    setEditingRule(rule)
    form.setFieldsValue({
      type: rule.type,
      severity: rule.severity,
      threshold: rule.threshold,
      duration: `${rule.duration / 60}m`,
      notifyMethods: rule.notifyMethods,
      enabled: rule.enabled,
    })
    setIsRuleModalOpen(true)
  }

  const handleDeleteRule = async (ruleId: string) => {
    try {
      await monitoringService.deleteAlertRule(ruleId)
      message.success('规则已删除')
      loadData()
    } catch (error) {
      message.error('删除规则失败')
    }
  }

  const handleSaveRule = async () => {
    try {
      const values = await form.validateFields()
      if (editingRule) {
        await monitoringService.updateAlertRule(editingRule.id, {
          threshold: values.threshold,
          duration: values.duration,
          enabled: values.enabled,
          notifyMethods: values.notifyMethods,
        })
        message.success('规则已更新')
      } else {
        await monitoringService.createAlertRule({
          environmentId,
          type: values.type,
          severity: values.severity,
          threshold: values.threshold,
          duration: values.duration,
          notifyMethods: values.notifyMethods,
        })
        message.success('规则已创建')
      }
      setIsRuleModalOpen(false)
      loadData()
    } catch (error) {
      message.error('保存规则失败')
    }
  }

  const alertColumns = [
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: AlertSeverity) => {
        const config = severityConfig[severity]
        return (
          <Tag color={config.color} icon={config.icon}>
            {config.text}
          </Tag>
        )
      },
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: AlertStatus) => {
        const config = statusConfig[status]
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: '时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (time: string) => new Date(time).toLocaleString('zh-CN'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: any, record: Alert) => (
        record.status === 'active' && (
          <Button
            type="link"
            size="small"
            icon={<CheckCircleOutlined />}
            onClick={() => handleAcknowledge(record.id)}
          >
            确认
          </Button>
        )
      ),
    },
  ]

  const ruleColumns = [
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => metricTypeOptions.find(o => o.value === type)?.label || type,
    },
    {
      title: '阈值',
      dataIndex: 'threshold',
      key: 'threshold',
      render: (threshold: number) => `${threshold}%`,
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: AlertSeverity) => {
        const config = severityConfig[severity]
        return <Tag color={config.color}>{config.text}</Tag>
      },
    },
    {
      title: '通知方式',
      dataIndex: 'notifyMethods',
      key: 'notifyMethods',
      render: (methods: string[]) => methods?.map(m => (
        <Tag key={m}>{m === 'web' ? '站内' : m === 'email' ? '邮件' : m}</Tag>
      )),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean) => (
        <Badge status={enabled ? 'success' : 'default'} text={enabled ? '启用' : '禁用'} />
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: AlertRule) => (
        <Space>
          <Tooltip title="编辑">
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEditRule(record)} />
          </Tooltip>
          <Popconfirm title="确定删除此规则？" onConfirm={() => handleDeleteRule(record.id)}>
            <Tooltip title="删除">
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const activeAlertCount = alerts.filter(a => a.status === 'active').length

  return (
    <div className="space-y-4">
      {/* Alerts */}
      <Card
        title={
          <Space>
            <BellOutlined />
            <span>告警列表</span>
            {activeAlertCount > 0 && (
              <Badge count={activeAlertCount} style={{ backgroundColor: '#ff4d4f' }} />
            )}
          </Space>
        }
        extra={
          <Button type="link" onClick={loadData}>
            刷新
          </Button>
        }
      >
        {alerts.length > 0 ? (
          <Table
            columns={alertColumns}
            dataSource={alerts}
            rowKey="id"
            loading={isLoading}
            pagination={{ pageSize: 10 }}
            size="small"
          />
        ) : (
          <Empty description="暂无告警" />
        )}
      </Card>

      {/* Alert Rules */}
      {showRules && (
        <Card
          title="告警规则"
          extra={
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreateRule}>
              添加规则
            </Button>
          }
        >
          {rules.length > 0 ? (
            <Table
              columns={ruleColumns}
              dataSource={rules}
              rowKey="id"
              loading={isLoading}
              pagination={false}
              size="small"
            />
          ) : (
            <Empty description="暂无告警规则" />
          )}
        </Card>
      )}

      {/* Rule Modal */}
      <Modal
        title={editingRule ? '编辑告警规则' : '创建告警规则'}
        open={isRuleModalOpen}
        onOk={handleSaveRule}
        onCancel={() => setIsRuleModalOpen(false)}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item name="type" label="监控类型" rules={[{ required: true }]}>
            <Select options={metricTypeOptions} disabled={!!editingRule} />
          </Form.Item>
          <Form.Item name="threshold" label="阈值 (%)" rules={[{ required: true }]}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="severity" label="严重程度" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'info', label: '信息' },
                { value: 'warning', label: '警告' },
                { value: 'critical', label: '严重' },
              ]}
              disabled={!!editingRule}
            />
          </Form.Item>
          <Form.Item name="duration" label="持续时间" rules={[{ required: true }]}>
            <Select
              options={[
                { value: '0s', label: '立即' },
                { value: '1m', label: '1 分钟' },
                { value: '5m', label: '5 分钟' },
                { value: '10m', label: '10 分钟' },
              ]}
            />
          </Form.Item>
          <Form.Item name="notifyMethods" label="通知方式" rules={[{ required: true }]}>
            <Select
              mode="multiple"
              options={[
                { value: 'web', label: '站内通知' },
                { value: 'email', label: '邮件通知' },
              ]}
            />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

// Generate mock alerts for demo
function generateMockAlerts(environmentId?: string): Alert[] {
  const now = new Date()
  return [
    {
      id: '1',
      environmentId: environmentId || 'env-1',
      userId: 'user-1',
      type: 'storage',
      severity: 'critical',
      status: 'active',
      title: '存储空间告警',
      message: '环境存储使用率已达 92.5%，超过阈值 90%',
      value: 92.5,
      threshold: 90,
      createdAt: new Date(now.getTime() - 30 * 60 * 1000).toISOString(),
      updatedAt: new Date(now.getTime() - 30 * 60 * 1000).toISOString(),
      notifyMethods: ['web', 'email'],
    },
    {
      id: '2',
      environmentId: environmentId || 'env-1',
      userId: 'user-1',
      type: 'memory',
      severity: 'warning',
      status: 'resolved',
      title: '内存使用告警',
      message: '环境内存使用率已达 85.2%，超过阈值 80%',
      value: 85.2,
      threshold: 80,
      createdAt: new Date(now.getTime() - 2 * 60 * 60 * 1000).toISOString(),
      updatedAt: new Date(now.getTime() - 60 * 60 * 1000).toISOString(),
      resolvedAt: new Date(now.getTime() - 60 * 60 * 1000).toISOString(),
      notifyMethods: ['web'],
    },
  ]
}

// Generate mock rules for demo
function generateMockRules(): AlertRule[] {
  const now = new Date()
  return [
    {
      id: '1',
      userId: 'user-1',
      type: 'storage',
      severity: 'critical',
      threshold: 90,
      duration: 0,
      enabled: true,
      notifyMethods: ['web', 'email'],
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    },
    {
      id: '2',
      userId: 'user-1',
      type: 'memory',
      severity: 'warning',
      threshold: 80,
      duration: 300,
      enabled: true,
      notifyMethods: ['web'],
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    },
    {
      id: '3',
      userId: 'user-1',
      type: 'cpu',
      severity: 'warning',
      threshold: 90,
      duration: 600,
      enabled: false,
      notifyMethods: ['web'],
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    },
  ]
}

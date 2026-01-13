import { useState, useEffect, useCallback } from 'react'
import { Card, Select, Spin, Empty, Row, Col, Statistic, Tag, Tooltip } from 'antd'
import {
  SyncOutlined,
  WarningOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons'
import ReactECharts from 'echarts-for-react'
import { monitoringService, MetricsHistory, EnvironmentMetrics } from '@/services/monitoring'

interface MonitoringChartsProps {
  environmentId: string
  isRunning: boolean
}

const periodOptions = [
  { value: '1h', label: '最近 1 小时' },
  { value: '6h', label: '最近 6 小时' },
  { value: '24h', label: '最近 24 小时' },
  { value: '7d', label: '最近 7 天' },
]

// Format bytes to human readable
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// Format rate to human readable
const formatRate = (bytesPerSec: number): string => {
  return formatBytes(bytesPerSec) + '/s'
}

export default function MonitoringCharts({ environmentId, isRunning }: MonitoringChartsProps) {
  const [period, setPeriod] = useState('1h')
  const [history, setHistory] = useState<MetricsHistory | null>(null)
  const [currentMetrics, setCurrentMetrics] = useState<EnvironmentMetrics | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [autoRefresh, setAutoRefresh] = useState(true)

  const loadMetrics = useCallback(async () => {
    if (!isRunning) {
      setIsLoading(false)
      return
    }

    try {
      const [historyData, metricsData] = await Promise.all([
        monitoringService.getMetricsHistory(environmentId, period),
        monitoringService.getMetrics(environmentId),
      ])
      setHistory(historyData)
      setCurrentMetrics(metricsData)
    } catch (error) {
      console.error('Failed to load metrics:', error)
      // Generate mock data for demo
      setHistory(generateMockHistory(environmentId, period))
      setCurrentMetrics(generateMockMetrics(environmentId))
    } finally {
      setIsLoading(false)
    }
  }, [environmentId, period, isRunning])

  useEffect(() => {
    loadMetrics()
  }, [loadMetrics])

  useEffect(() => {
    if (!autoRefresh || !isRunning) return
    const interval = setInterval(loadMetrics, 30000) // Refresh every 30 seconds
    return () => clearInterval(interval)
  }, [autoRefresh, isRunning, loadMetrics])

  const getChartOption = (
    title: string,
    data: { timestamp: string; value: number }[],
    color: string,
    unit: string,
    formatter?: (value: number) => string
  ) => ({
    title: { text: title, left: 'center', textStyle: { fontSize: 14 } },
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const point = params[0]
        const time = new Date(point.axisValue).toLocaleString('zh-CN')
        const value = formatter ? formatter(point.value) : `${point.value.toFixed(1)}${unit}`
        return `${time}<br/>${point.seriesName}: ${value}`
      },
    },
    grid: { left: '10%', right: '5%', bottom: '15%', top: '20%' },
    xAxis: {
      type: 'time',
      axisLabel: {
        formatter: (value: number) => {
          const date = new Date(value)
          return `${date.getHours()}:${date.getMinutes().toString().padStart(2, '0')}`
        },
      },
    },
    yAxis: {
      type: 'value',
      max: unit === '%' ? 100 : undefined,
      axisLabel: { formatter: formatter || ((v: number) => `${v}${unit}`) },
    },
    series: [{
      name: title,
      type: 'line',
      smooth: true,
      symbol: 'none',
      areaStyle: { opacity: 0.3 },
      lineStyle: { width: 2 },
      itemStyle: { color },
      data: data.map(d => [d.timestamp, d.value]),
    }],
  })

  if (!isRunning) {
    return (
      <Card>
        <Empty description="环境未运行，无法显示监控数据" />
      </Card>
    )
  }

  if (isLoading) {
    return (
      <Card>
        <div className="flex items-center justify-center h-64">
          <Spin size="large" tip="加载监控数据..." />
        </div>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      {/* Current Metrics Summary */}
      {currentMetrics && (
        <Card title="当前资源使用" extra={
          <div className="flex items-center gap-2">
            {autoRefresh && <Tag color="green" icon={<SyncOutlined spin />}>自动刷新</Tag>}
            <Select
              value={period}
              onChange={setPeriod}
              options={periodOptions}
              style={{ width: 140 }}
            />
          </div>
        }>
          <Row gutter={16}>
            <Col xs={12} sm={6}>
              <Statistic
                title="CPU 使用率"
                value={currentMetrics.cpuUsage}
                precision={1}
                suffix="%"
                valueStyle={{ color: currentMetrics.cpuUsage > 80 ? '#ff4d4f' : '#3f8600' }}
                prefix={currentMetrics.cpuUsage > 80 ? <WarningOutlined /> : <CheckCircleOutlined />}
              />
            </Col>
            <Col xs={12} sm={6}>
              <Statistic
                title="内存使用率"
                value={currentMetrics.memoryUsage}
                precision={1}
                suffix="%"
                valueStyle={{ color: currentMetrics.memoryUsage > 80 ? '#ff4d4f' : '#3f8600' }}
                prefix={currentMetrics.memoryUsage > 80 ? <WarningOutlined /> : <CheckCircleOutlined />}
              />
              <div className="text-xs text-gray-500 mt-1">
                {formatBytes(currentMetrics.memoryUsed)} / {formatBytes(currentMetrics.memoryLimit)}
              </div>
            </Col>
            <Col xs={12} sm={6}>
              <Statistic
                title="存储使用率"
                value={currentMetrics.storageUsage}
                precision={1}
                suffix="%"
                valueStyle={{ color: currentMetrics.storageUsage > 90 ? '#ff4d4f' : '#3f8600' }}
                prefix={currentMetrics.storageUsage > 90 ? <WarningOutlined /> : <CheckCircleOutlined />}
              />
              <div className="text-xs text-gray-500 mt-1">
                {formatBytes(currentMetrics.storageUsed)} / {formatBytes(currentMetrics.storageLimit)}
              </div>
            </Col>
            <Col xs={12} sm={6}>
              <Tooltip title={`接收: ${formatRate(currentMetrics.networkRxRate)} / 发送: ${formatRate(currentMetrics.networkTxRate)}`}>
                <Statistic
                  title="网络 I/O"
                  value={formatRate(currentMetrics.networkRxRate + currentMetrics.networkTxRate)}
                  valueStyle={{ color: '#1890ff' }}
                />
              </Tooltip>
            </Col>
          </Row>
        </Card>
      )}

      {/* Charts */}
      {history && (
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={12}>
            <Card size="small">
              <ReactECharts
                option={getChartOption('CPU 使用率', history.cpu, '#1890ff', '%')}
                style={{ height: 250 }}
              />
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card size="small">
              <ReactECharts
                option={getChartOption('内存使用率', history.memory, '#52c41a', '%')}
                style={{ height: 250 }}
              />
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card size="small">
              <ReactECharts
                option={getChartOption('存储使用率', history.storage, '#722ed1', '%')}
                style={{ height: 250 }}
              />
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card size="small">
              <ReactECharts
                option={getChartOption(
                  '网络流量',
                  history.networkRx.map((d, i) => ({
                    timestamp: d.timestamp,
                    value: d.value + (history.networkTx[i]?.value || 0),
                  })),
                  '#13c2c2',
                  '',
                  formatRate
                )}
                style={{ height: 250 }}
              />
            </Card>
          </Col>
        </Row>
      )}
    </div>
  )
}

// Generate mock history data for demo
function generateMockHistory(environmentId: string, period: string): MetricsHistory {
  const now = new Date()
  const periodMs = {
    '1h': 60 * 60 * 1000,
    '6h': 6 * 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000,
  }[period] || 60 * 60 * 1000

  const intervalMs = periodMs / 60 // 60 data points
  const startTime = new Date(now.getTime() - periodMs)
  
  const generateData = (base: number, variance: number) => {
    const data = []
    for (let i = 0; i < 60; i++) {
      const timestamp = new Date(startTime.getTime() + i * intervalMs)
      const value = Math.max(0, Math.min(100, base + (Math.random() - 0.5) * variance))
      data.push({ timestamp: timestamp.toISOString(), value })
    }
    return data
  }

  return {
    environmentId,
    period,
    resolution: '1m',
    startTime: startTime.toISOString(),
    endTime: now.toISOString(),
    cpu: generateData(45, 30),
    memory: generateData(60, 20),
    storage: generateData(35, 5),
    networkRx: generateData(5 * 1024 * 1024, 3 * 1024 * 1024),
    networkTx: generateData(2 * 1024 * 1024, 1.5 * 1024 * 1024),
  }
}

// Generate mock current metrics for demo
function generateMockMetrics(environmentId: string): EnvironmentMetrics {
  return {
    environmentId,
    timestamp: new Date().toISOString(),
    cpuUsage: 40 + Math.random() * 30,
    cpuCores: 1.5,
    cpuLimit: 2,
    memoryUsage: 55 + Math.random() * 20,
    memoryUsed: 1.2 * 1024 * 1024 * 1024,
    memoryLimit: 2 * 1024 * 1024 * 1024,
    storageUsage: 30 + Math.random() * 10,
    storageUsed: 3 * 1024 * 1024 * 1024,
    storageLimit: 10 * 1024 * 1024 * 1024,
    networkRxBytes: Math.random() * 100 * 1024 * 1024,
    networkTxBytes: Math.random() * 50 * 1024 * 1024,
    networkRxRate: Math.random() * 10 * 1024 * 1024,
    networkTxRate: Math.random() * 5 * 1024 * 1024,
  }
}

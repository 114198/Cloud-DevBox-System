/**
 * CollaborationInvite Component
 * Handles invitation link generation and permission settings for collaboration
 * Requirements: 4.4 - Project permission management with owner, editor, viewer roles
 */

import { useState, useEffect } from 'react'
import {
  Modal,
  Form,
  Input,
  Select,
  Button,
  Space,
  Typography,
  Divider,
  List,
  Tag,
  Tooltip,
  message,
  Spin,
  Alert,
  InputNumber,
  Card,
  Popconfirm,
  Empty,
} from 'antd'
import {
  LinkOutlined,
  CopyOutlined,
  DeleteOutlined,
  UserAddOutlined,
  ClockCircleOutlined,
  TeamOutlined,
  CheckCircleOutlined,
  MailOutlined,
  QrcodeOutlined,
} from '@ant-design/icons'
import { collaborationService, InviteLink } from '@/services/collaboration'
import { useCollaborationStore } from '@/stores/collaboration'
// Note: Install qrcode.react for QR code support: npm install qrcode.react @types/qrcode.react

const { Text, Paragraph } = Typography
const { Option } = Select

interface CollaborationInviteProps {
  visible: boolean
  onClose: () => void
  environmentId: string
  environmentName: string
}

interface InviteFormValues {
  role: 'participant' | 'viewer'
  expiresInHours: number
  maxUses: number
}

export default function CollaborationInvite({
  visible,
  onClose,
  environmentId,
  environmentName,
}: CollaborationInviteProps) {
  const [form] = Form.useForm<InviteFormValues>()
  const [loading, setLoading] = useState(false)
  const [generatedLink, setGeneratedLink] = useState<string | null>(null)
  const [activeInvites, setActiveInvites] = useState<InviteLink[]>([])
  const [loadingInvites, setLoadingInvites] = useState(false)
  const [showQRCode, setShowQRCode] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const { sessionId } = useCollaborationStore()

  // Load active invites when modal opens
  useEffect(() => {
    if (visible && sessionId) {
      loadActiveInvites()
    }
  }, [visible, sessionId])

  const loadActiveInvites = async () => {
    if (!sessionId) return

    setLoadingInvites(true)
    try {
      const invites = await collaborationService.getSessionInvites(sessionId)
      setActiveInvites(invites.filter(inv => new Date(inv.expiresAt) > new Date()))
    } catch (err) {
      console.error('Failed to load invites:', err)
    } finally {
      setLoadingInvites(false)
    }
  }

  // Generate invite link
  const handleGenerateLink = async (values: InviteFormValues) => {
    if (!sessionId) {
      message.error('未连接到协作会话')
      return
    }

    setLoading(true)
    try {
      const invite = await collaborationService.createInviteLink(
        sessionId,
        environmentId,
        values.role,
        values.expiresInHours,
        values.maxUses
      )

      const link = `${window.location.origin}/collaboration/join/${invite.token}`
      setGeneratedLink(link)
      setActiveInvites((prev: InviteLink[]) => [invite, ...prev])
      message.success('邀请链接已生成')
    } catch (err) {
      message.error('生成邀请链接失败')
    } finally {
      setLoading(false)
    }
  }

  // Copy link to clipboard
  const handleCopyLink = async (link: string) => {
    try {
      await navigator.clipboard.writeText(link)
      message.success('链接已复制到剪贴板')
    } catch {
      message.error('复制失败，请手动复制')
    }
  }

  // Revoke invite
  const handleRevokeInvite = async (inviteId: string) => {
    setDeletingId(inviteId)
    try {
      await collaborationService.revokeInviteLink(inviteId)
      setActiveInvites((prev: InviteLink[]) => prev.filter((inv: InviteLink) => inv.id !== inviteId))
      message.success('邀请链接已撤销')
    } catch (err) {
      message.error('撤销失败')
    } finally {
      setDeletingId(null)
    }
  }

  // Format expiration time
  const formatExpiration = (expiresAt: string) => {
    const expires = new Date(expiresAt)
    const now = new Date()
    const diff = expires.getTime() - now.getTime()

    if (diff <= 0) return '已过期'
    if (diff < 3600000) return `${Math.ceil(diff / 60000)} 分钟后过期`
    if (diff < 86400000) return `${Math.ceil(diff / 3600000)} 小时后过期`
    return `${Math.ceil(diff / 86400000)} 天后过期`
  }

  // Get role tag
  const getRoleTag = (role: string) => {
    const config: Record<string, { color: string; label: string }> = {
      participant: { color: 'blue', label: '编辑者' },
      viewer: { color: 'default', label: '查看者' },
    }
    const { color, label } = config[role] || config.viewer
    return <Tag color={color}>{label}</Tag>
  }

  // Build invite link from token
  const getInviteLink = (token: string) => {
    return `${window.location.origin}/collaboration/join/${token}`
  }

  return (
    <Modal
      title={
        <Space>
          <UserAddOutlined />
          邀请协作者
        </Space>
      }
      open={visible}
      onCancel={onClose}
      footer={null}
      width={600}
      destroyOnClose
    >
      <div className="collaboration-invite">
        {/* Environment info */}
        <Alert
          message={
            <Space>
              <TeamOutlined />
              <Text>正在邀请协作者加入环境：</Text>
              <Text strong>{environmentName}</Text>
            </Space>
          }
          type="info"
          showIcon={false}
          style={{ marginBottom: 16 }}
        />

        {/* Generate new invite form */}
        <Card
          title="生成邀请链接"
          size="small"
          style={{ marginBottom: 16 }}
        >
          <Form
            form={form}
            layout="vertical"
            onFinish={handleGenerateLink}
            initialValues={{
              role: 'participant',
              expiresInHours: 24,
              maxUses: 10,
            }}
          >
            <Form.Item
              name="role"
              label="权限角色"
              tooltip="编辑者可以编辑代码，查看者只能查看"
            >
              <Select>
                <Option value="participant">
                  <Space>
                    <Tag color="blue">编辑者</Tag>
                    <Text type="secondary">可以编辑代码和文件</Text>
                  </Space>
                </Option>
                <Option value="viewer">
                  <Space>
                    <Tag>查看者</Tag>
                    <Text type="secondary">只能查看，不能编辑</Text>
                  </Space>
                </Option>
              </Select>
            </Form.Item>

            <Space style={{ width: '100%' }} size="large">
              <Form.Item
                name="expiresInHours"
                label="有效期"
                style={{ marginBottom: 0, flex: 1 }}
              >
                <Select style={{ width: 120 }}>
                  <Option value={1}>1 小时</Option>
                  <Option value={6}>6 小时</Option>
                  <Option value={24}>24 小时</Option>
                  <Option value={72}>3 天</Option>
                  <Option value={168}>7 天</Option>
                  <Option value={720}>30 天</Option>
                </Select>
              </Form.Item>

              <Form.Item
                name="maxUses"
                label="最大使用次数"
                style={{ marginBottom: 0, flex: 1 }}
              >
                <InputNumber min={1} max={100} style={{ width: 120 }} />
              </Form.Item>
            </Space>

            <Form.Item style={{ marginTop: 16, marginBottom: 0 }}>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                icon={<LinkOutlined />}
                block
              >
                生成邀请链接
              </Button>
            </Form.Item>
          </Form>

          {/* Generated link display */}
          {generatedLink && (
            <div style={{ marginTop: 16 }}>
              <Divider style={{ margin: '12px 0' }} />
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                  padding: '8px 12px',
                  background: '#f5f5f5',
                  borderRadius: 6,
                }}
              >
                <CheckCircleOutlined style={{ color: '#52c41a' }} />
                <Input
                  value={generatedLink}
                  readOnly
                  style={{ flex: 1 }}
                />
                <Tooltip title="复制链接">
                  <Button
                    icon={<CopyOutlined />}
                    onClick={() => handleCopyLink(generatedLink)}
                  />
                </Tooltip>
                <Tooltip title="显示二维码">
                  <Button
                    icon={<QrcodeOutlined />}
                    onClick={() => setShowQRCode(!showQRCode)}
                  />
                </Tooltip>
              </div>

              {/* QR Code */}
              {showQRCode && (
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'center',
                    marginTop: 16,
                    padding: 16,
                    background: '#fff',
                    borderRadius: 8,
                    border: '1px solid #f0f0f0',
                  }}
                >
                  <QRCodeSVG
                    value={generatedLink}
                    size={160}
                    level="M"
                    includeMargin
                  />
                </div>
              )}

              <Paragraph
                type="secondary"
                style={{ marginTop: 8, marginBottom: 0, fontSize: 12 }}
              >
                将此链接分享给协作者，他们可以通过链接加入协作会话
              </Paragraph>
            </div>
          )}
        </Card>

        {/* Active invites list */}
        <Card
          title={
            <Space>
              <span>活跃的邀请链接</span>
              {activeInvites.length > 0 && (
                <Tag>{activeInvites.length}</Tag>
              )}
            </Space>
          }
          size="small"
          extra={
            <Button
              type="link"
              size="small"
              onClick={loadActiveInvites}
              loading={loadingInvites}
            >
              刷新
            </Button>
          }
        >
          {loadingInvites ? (
            <div style={{ textAlign: 'center', padding: 24 }}>
              <Spin />
            </div>
          ) : activeInvites.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="暂无活跃的邀请链接"
            />
          ) : (
            <List
              dataSource={activeInvites}
              renderItem={(invite: InviteLink) => (
                <List.Item
                  key={invite.id}
                  actions={[
                    <Tooltip title="复制链接" key="copy">
                      <Button
                        type="text"
                        size="small"
                        icon={<CopyOutlined />}
                        onClick={() => handleCopyLink(getInviteLink(invite.token))}
                      />
                    </Tooltip>,
                    <Popconfirm
                      key="revoke"
                      title="确定要撤销此邀请链接吗？"
                      description="撤销后，使用此链接的用户将无法加入"
                      onConfirm={() => handleRevokeInvite(invite.id)}
                      okText="撤销"
                      cancelText="取消"
                      okButtonProps={{ danger: true }}
                    >
                      <Tooltip title="撤销">
                        <Button
                          type="text"
                          size="small"
                          danger
                          icon={<DeleteOutlined />}
                          loading={deletingId === invite.id}
                        />
                      </Tooltip>
                    </Popconfirm>,
                  ]}
                >
                  <List.Item.Meta
                    title={
                      <Space>
                        {getRoleTag(invite.role)}
                        <Text copyable={{ text: getInviteLink(invite.token) }}>
                          {invite.token.substring(0, 8)}...
                        </Text>
                      </Space>
                    }
                    description={
                      <Space split={<Divider type="vertical" />}>
                        <Text type="secondary">
                          <ClockCircleOutlined /> {formatExpiration(invite.expiresAt)}
                        </Text>
                        <Text type="secondary">
                          已使用 {invite.usedCount}/{invite.maxUses} 次
                        </Text>
                      </Space>
                    }
                  />
                </List.Item>
              )}
            />
          )}
        </Card>

        {/* Email invite section (optional) */}
        <Divider>或通过邮件邀请</Divider>
        <Form layout="inline" style={{ justifyContent: 'center' }}>
          <Form.Item>
            <Input
              placeholder="输入邮箱地址"
              prefix={<MailOutlined />}
              style={{ width: 300 }}
            />
          </Form.Item>
          <Form.Item>
            <Button icon={<MailOutlined />}>
              发送邀请
            </Button>
          </Form.Item>
        </Form>
        <Text
          type="secondary"
          style={{ display: 'block', textAlign: 'center', marginTop: 8, fontSize: 12 }}
        >
          邀请邮件将包含加入协作的链接
        </Text>
      </div>
    </Modal>
  )
}

// Simple QR Code component fallback if qrcode.react is not installed
function QRCodeSVG({ value: _value, size, level: _level, includeMargin: _includeMargin }: {
  value: string
  size: number
  level: string
  includeMargin: boolean
}) {
  // This is a placeholder - in production, use the actual qrcode.react library
  return (
    <div
      style={{
        width: size,
        height: size,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f5f5f5',
        borderRadius: 8,
      }}
    >
      <Text type="secondary" style={{ fontSize: 12, textAlign: 'center' }}>
        二维码<br />
        (需安装 qrcode.react)
      </Text>
    </div>
  )
}

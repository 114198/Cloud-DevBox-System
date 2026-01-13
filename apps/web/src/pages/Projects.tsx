import { Table, Button, Card, Space, Tag } from 'antd'
import { PlusOutlined, TeamOutlined, FolderOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'

interface Project {
  id: string
  name: string
  description: string
  environmentCount: number
  memberCount: number
  createdAt: string
}

const mockProjects: Project[] = [
  {
    id: '1',
    name: 'E-commerce Platform',
    description: '电商平台项目',
    environmentCount: 3,
    memberCount: 5,
    createdAt: '2024-01-10',
  },
  {
    id: '2',
    name: 'Mobile App Backend',
    description: '移动应用后端服务',
    environmentCount: 2,
    memberCount: 3,
    createdAt: '2024-01-08',
  },
]

export default function Projects() {
  const columns: ColumnsType<Project> = [
    {
      title: '项目名称',
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <div>
          <div className="font-medium">{text}</div>
          <div className="text-gray-500 text-sm">{record.description}</div>
        </div>
      ),
    },
    {
      title: '环境数',
      dataIndex: 'environmentCount',
      key: 'environmentCount',
      render: (count) => (
        <Tag icon={<FolderOutlined />} color="blue">
          {count} 个环境
        </Tag>
      ),
    },
    {
      title: '成员数',
      dataIndex: 'memberCount',
      key: 'memberCount',
      render: (count) => (
        <Tag icon={<TeamOutlined />} color="green">
          {count} 人
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space size="small">
          <Button type="link">查看</Button>
          <Button type="link">设置</Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold">项目管理</h1>
        <Button type="primary" icon={<PlusOutlined />}>
          创建项目
        </Button>
      </div>

      <Card>
        <Table columns={columns} dataSource={mockProjects} rowKey="id" />
      </Card>
    </div>
  )
}

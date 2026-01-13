import { useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Avatar, Dropdown, Button } from 'antd'
import {
  DashboardOutlined,
  CloudServerOutlined,
  AppstoreOutlined,
  ProjectOutlined,
  SettingOutlined,
  UserOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  GithubOutlined,
  WalletOutlined,
  CloudUploadOutlined,
} from '@ant-design/icons'
import { useAuth } from '@/hooks/useAuth'

const { Header, Sider, Content } = Layout

const menuItems = [
  {
    key: '/',
    icon: <DashboardOutlined />,
    label: '仪表盘',
  },
  {
    key: '/environments',
    icon: <CloudServerOutlined />,
    label: '开发环境',
  },
  {
    key: '/templates',
    icon: <AppstoreOutlined />,
    label: '环境模板',
  },
  {
    key: '/projects',
    icon: <ProjectOutlined />,
    label: '项目管理',
  },
  {
    key: '/git',
    icon: <GithubOutlined />,
    label: 'Git 仓库',
  },
  {
    key: '/billing',
    icon: <WalletOutlined />,
    label: '账单',
  },
  {
    key: '/backup',
    icon: <CloudUploadOutlined />,
    label: '备份管理',
  },
  {
    key: '/settings',
    icon: <SettingOutlined />,
    label: '设置',
  },
]

export default function MainLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout, isLoading } = useAuth()

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(key)
  }

  const userMenuItems = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: '个人资料',
      onClick: () => navigate('/settings'),
    },
    {
      type: 'divider' as const,
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      danger: true,
      disabled: isLoading,
      onClick: logout,
    },
  ]

  return (
    <Layout className="min-h-screen">
      <Sider trigger={null} collapsible collapsed={collapsed} theme="light">
        <div className="h-16 flex items-center justify-center border-b">
          <h1 className={`font-bold text-primary-600 ${collapsed ? 'text-lg' : 'text-xl'}`}>
            {collapsed ? 'DB' : 'Cloud DevBox'}
          </h1>
        </div>
        <Menu
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={handleMenuClick}
          className="border-r-0"
        />
      </Sider>
      <Layout>
        <Header className="bg-white px-4 flex items-center justify-between border-b">
          <Button
            type="text"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={() => setCollapsed(!collapsed)}
          />
          <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
            <div className="flex items-center cursor-pointer hover:bg-gray-50 px-3 py-1 rounded">
              <Avatar 
                icon={<UserOutlined />} 
                src={user?.avatar}
                className="mr-2" 
              />
              <span className="text-gray-700">{user?.displayName || user?.username || '用户'}</span>
            </div>
          </Dropdown>
        </Header>
        <Content className="m-4 p-6 bg-white rounded-lg min-h-[280px]">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

import { Card, Row, Col, Tag, Button, Input, Select, Spin, Empty, Pagination, message } from 'antd'
import { SearchOutlined, PlusOutlined, ReloadOutlined, FilterOutlined } from '@ant-design/icons'
import { useState, useEffect, useCallback, ChangeEvent } from 'react'
import { useTemplateStore } from '@/stores/template'
import { templateService, Template } from '@/services/template'
import TemplateConfigModal from '@/components/TemplateConfigModal'

// Category icons mapping
const categoryIcons: Record<string, string> = {
  '前端': '🎨',
  '后端': '⚙️',
  '全栈': '🔄',
  '数据库': '🗄️',
  'DevOps': '🚀',
  '其他': '📦',
}

// Template icon mapping based on technology
const getTemplateIcon = (template: Template): string => {
  if (template.icon) return template.icon
  
  const name = template.name.toLowerCase()
  if (name.includes('react')) return '⚛️'
  if (name.includes('vue')) return '💚'
  if (name.includes('angular')) return '🅰️'
  if (name.includes('next')) return '▲'
  if (name.includes('nuxt')) return '💚'
  if (name.includes('node')) return '🟢'
  if (name.includes('python') || name.includes('django') || name.includes('fastapi')) return '🐍'
  if (name.includes('go') || name.includes('gin')) return '🐹'
  if (name.includes('rust') || name.includes('axum')) return '🦀'
  if (name.includes('java') || name.includes('spring')) return '☕'
  if (name.includes('postgres')) return '🐘'
  if (name.includes('mysql')) return '🐬'
  if (name.includes('mongo')) return '🍃'
  if (name.includes('redis')) return '🔴'
  
  return categoryIcons[template.category] || '📦'
}

// Tag color mapping
const getTagColor = (tag: string): string => {
  const tagLower = tag.toLowerCase()
  if (tagLower.includes('react') || tagLower.includes('vue') || tagLower.includes('angular')) return 'blue'
  if (tagLower.includes('typescript')) return 'cyan'
  if (tagLower.includes('javascript')) return 'gold'
  if (tagLower.includes('python')) return 'green'
  if (tagLower.includes('go')) return 'geekblue'
  if (tagLower.includes('rust')) return 'orange'
  if (tagLower.includes('java')) return 'red'
  if (tagLower.includes('node')) return 'lime'
  if (tagLower.includes('docker') || tagLower.includes('k8s')) return 'purple'
  return 'default'
}

export default function Templates() {
  const {
    templates,
    total,
    page,
    pageSize,
    totalPages,
    isLoading,
    error,
    filters,
    selectedTemplate,
    setTemplates,
    setLoading,
    setError,
    setFilters,
    resetFilters,
    selectTemplate,
    setPage,
  } = useTemplateStore()

  const [categories, setCategories] = useState<string[]>([])
  const [allTags, setAllTags] = useState<string[]>([])
  const [configModalVisible, setConfigModalVisible] = useState(false)

  // Fetch templates
  const fetchTemplates = useCallback(async () => {
    setLoading(true)
    try {
      const result = await templateService.list(filters)
      setTemplates(result.data || [], result.total, result.page, result.page_size, result.total_pages)
    } catch (err) {
      console.error('Failed to fetch templates:', err)
      setError('加载模板列表失败，请稍后重试')
      message.error('加载模板列表失败')
    }
  }, [filters, setTemplates, setLoading, setError])

  // Fetch categories and tags
  const fetchMetadata = useCallback(async () => {
    try {
      const [categoriesData, tagsData] = await Promise.all([
        templateService.getValidCategories(),
        templateService.getTags(),
      ])
      setCategories(categoriesData)
      setAllTags(tagsData)
    } catch (err) {
      console.error('Failed to fetch metadata:', err)
    }
  }, [])

  useEffect(() => {
    fetchTemplates()
  }, [fetchTemplates])

  useEffect(() => {
    fetchMetadata()
  }, [fetchMetadata])

  // Handle search
  const handleSearch = (value: string) => {
    setFilters({ search: value || undefined })
  }

  // Handle category change
  const handleCategoryChange = (value: string | undefined) => {
    setFilters({ category: value })
  }

  // Handle tag filter
  const handleTagsChange = (values: string[]) => {
    setFilters({ tags: values.length > 0 ? values : undefined })
  }

  // Handle page change
  const handlePageChange = (newPage: number, newPageSize?: number) => {
    if (newPageSize && newPageSize !== pageSize) {
      setFilters({ page: 1, page_size: newPageSize })
    } else {
      setPage(newPage)
    }
  }

  // Handle template selection for configuration
  const handleUseTemplate = (template: Template) => {
    selectTemplate(template)
    setConfigModalVisible(true)
  }

  // Handle modal close
  const handleConfigModalClose = () => {
    setConfigModalVisible(false)
    selectTemplate(null)
  }

  // Handle refresh
  const handleRefresh = () => {
    fetchTemplates()
    fetchMetadata()
  }

  // Handle reset filters
  const handleResetFilters = () => {
    resetFilters()
  }

  return (
    <div>
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold">环境模板</h1>
        <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={isLoading}>
          刷新
        </Button>
      </div>

      {/* Filters */}
      <div className="bg-white p-4 rounded-lg shadow-sm mb-6">
        <div className="flex flex-wrap gap-4 items-center">
          <Input
            placeholder="搜索模板名称或描述..."
            prefix={<SearchOutlined />}
            value={filters.search || ''}
            onChange={(e: ChangeEvent<HTMLInputElement>) => handleSearch(e.target.value)}
            className="w-64"
            allowClear
          />
          
          <Select
            placeholder="选择分类"
            allowClear
            value={filters.category}
            onChange={handleCategoryChange}
            className="w-32"
          >
            {categories.map((cat: string) => (
              <Select.Option key={cat} value={cat}>
                {categoryIcons[cat] || '📦'} {cat}
              </Select.Option>
            ))}
          </Select>

          <Select
            mode="multiple"
            placeholder="筛选标签"
            allowClear
            value={filters.tags || []}
            onChange={handleTagsChange}
            className="min-w-48"
            maxTagCount={2}
          >
            {allTags.map((tag: string) => (
              <Select.Option key={tag} value={tag}>
                {tag}
              </Select.Option>
            ))}
          </Select>

          <Button 
            icon={<FilterOutlined />} 
            onClick={handleResetFilters}
            disabled={!filters.search && !filters.category && !filters.tags?.length}
          >
            重置筛选
          </Button>
        </div>

        {/* Active filters display */}
        {(filters.search || filters.category || filters.tags?.length) && (
          <div className="mt-3 flex items-center gap-2 text-sm text-gray-500">
            <span>当前筛选:</span>
            {filters.search && (
              <Tag closable onClose={() => setFilters({ search: undefined })}>
                搜索: {filters.search}
              </Tag>
            )}
            {filters.category && (
              <Tag closable onClose={() => setFilters({ category: undefined })} color="blue">
                分类: {filters.category}
              </Tag>
            )}
            {filters.tags?.map((tag: string) => (
              <Tag
                key={tag}
                closable
                onClose={() => setFilters({ tags: filters.tags?.filter((t: string) => t !== tag) })}
                color="green"
              >
                {tag}
              </Tag>
            ))}
          </div>
        )}
      </div>

      {/* Loading state */}
      {isLoading && (
        <div className="flex justify-center items-center py-20">
          <Spin size="large" tip="加载模板中..." />
        </div>
      )}

      {/* Error state */}
      {error && !isLoading && (
        <div className="flex justify-center items-center py-20">
          <Empty
            description={error}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button type="primary" onClick={handleRefresh}>
              重新加载
            </Button>
          </Empty>
        </div>
      )}

      {/* Empty state */}
      {!isLoading && !error && templates.length === 0 && (
        <div className="flex justify-center items-center py-20">
          <Empty
            description={
              filters.search || filters.category || filters.tags?.length
                ? '没有找到匹配的模板，请尝试调整筛选条件'
                : '暂无可用模板'
            }
          >
            {(filters.search || filters.category || filters.tags?.length) && (
              <Button type="primary" onClick={handleResetFilters}>
                清除筛选
              </Button>
            )}
          </Empty>
        </div>
      )}

      {/* Template grid */}
      {!isLoading && !error && templates.length > 0 && (
        <>
          <Row gutter={[16, 16]}>
            {templates.map((template: Template) => (
              <Col key={template.id} xs={24} sm={12} md={8} lg={6}>
                <Card
                  hoverable
                  className="h-full flex flex-col"
                  actions={[
                    <Button
                      type="primary"
                      icon={<PlusOutlined />}
                      key="create"
                      onClick={() => handleUseTemplate(template)}
                    >
                      使用模板
                    </Button>,
                  ]}
                >
                  <div className="flex items-start gap-3 mb-3">
                    <div className="text-4xl">{getTemplateIcon(template)}</div>
                    <div className="flex-1 min-w-0">
                      <h3 className="font-semibold text-lg truncate" title={template.display_name}>
                        {template.display_name}
                      </h3>
                      <span className="text-xs text-gray-400">v{template.version}</span>
                    </div>
                  </div>
                  
                  <p className="text-gray-500 text-sm mb-3 line-clamp-2" title={template.description}>
                    {template.description || '暂无描述'}
                  </p>

                  {/* Runtime info */}
                  {template.runtime && (
                    <div className="text-xs text-gray-400 mb-2">
                      {template.runtime.language} {template.runtime.version}
                      {template.runtime.framework && ` + ${template.runtime.framework}`}
                    </div>
                  )}

                  {/* Tags */}
                  <div className="flex flex-wrap gap-1">
                    <Tag color="purple">{template.category}</Tag>
                    {template.tags?.slice(0, 3).map((tag: string) => (
                      <Tag key={tag} color={getTagColor(tag)}>
                        {tag}
                      </Tag>
                    ))}
                    {template.tags && template.tags.length > 3 && (
                      <Tag>+{template.tags.length - 3}</Tag>
                    )}
                  </div>
                </Card>
              </Col>
            ))}
          </Row>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex justify-center mt-6">
              <Pagination
                current={page}
                pageSize={pageSize}
                total={total}
                onChange={handlePageChange}
                showSizeChanger
                showQuickJumper
                showTotal={(total: number) => `共 ${total} 个模板`}
                pageSizeOptions={['12', '20', '40', '60']}
              />
            </div>
          )}
        </>
      )}

      {/* Template Configuration Modal */}
      <TemplateConfigModal
        visible={configModalVisible}
        template={selectedTemplate}
        onClose={handleConfigModalClose}
      />
    </div>
  )
}

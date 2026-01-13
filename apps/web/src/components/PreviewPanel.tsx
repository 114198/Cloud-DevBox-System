import React, { useState, useEffect } from 'react';
import {
  Domain,
  ShareLink,
  HotReloadConfig,
  domainApi,
  shareLinkApi,
  hotReloadApi,
  getDurationLabel,
  getStatusColor,
  formatExpiresAt,
} from '../services/preview';

interface PreviewPanelProps {
  environmentId: string;
}

export const PreviewPanel: React.FC<PreviewPanelProps> = ({ environmentId }) => {
  const [domains, setDomains] = useState<Domain[]>([]);
  const [shareLinks, setShareLinks] = useState<ShareLink[]>([]);
  const [hotReloadConfig, setHotReloadConfig] = useState<HotReloadConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'domains' | 'share' | 'hotreload'>('domains');

  // Create domain form state
  const [showCreateDomain, setShowCreateDomain] = useState(false);
  const [newDomainPort, setNewDomainPort] = useState(3000);
  const [newDomainType, setNewDomainType] = useState<'Auto' | 'Custom'>('Auto');
  const [newCustomDomain, setNewCustomDomain] = useState('');

  // Create share link form state
  const [showCreateShareLink, setShowCreateShareLink] = useState(false);
  const [selectedDomainId, setSelectedDomainId] = useState('');
  const [shareLinkDuration, setShareLinkDuration] = useState<string>('24h');
  const [shareLinkPassword, setShareLinkPassword] = useState('');
  const [shareLinkMaxViews, setShareLinkMaxViews] = useState<number | undefined>(undefined);

  useEffect(() => {
    loadData();
  }, [environmentId]);

  const loadData = async () => {
    try {
      setLoading(true);
      setError(null);

      const [domainsRes, shareLinksRes, configRes] = await Promise.all([
        domainApi.list({ environmentId }),
        shareLinkApi.list({ environmentId }),
        hotReloadApi.getConfig(environmentId),
      ]);

      setDomains(domainsRes.domains);
      setShareLinks(shareLinksRes.shareLinks);
      setHotReloadConfig(configRes);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load preview data');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateDomain = async () => {
    try {
      const domain = await domainApi.create({
        environmentId,
        port: newDomainPort,
        type: newDomainType,
        customDomain: newDomainType === 'Custom' ? newCustomDomain : undefined,
      });
      setDomains([...domains, domain]);
      setShowCreateDomain(false);
      setNewDomainPort(3000);
      setNewDomainType('Auto');
      setNewCustomDomain('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create domain');
    }
  };

  const handleDeleteDomain = async (id: string) => {
    if (!confirm('Are you sure you want to delete this domain?')) return;
    try {
      await domainApi.delete(id);
      setDomains(domains.filter(d => d.id !== id));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete domain');
    }
  };

  const handleVerifyDomain = async (id: string) => {
    try {
      const result = await domainApi.verify(id);
      if (result.valid) {
        loadData(); // Reload to get updated status
      } else {
        setError(result.errors?.join(', ') || 'Domain verification failed');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to verify domain');
    }
  };

  const handleCreateShareLink = async () => {
    try {
      const shareLink = await shareLinkApi.create({
        domainId: selectedDomainId,
        duration: shareLinkDuration as any,
        password: shareLinkPassword || undefined,
        maxViews: shareLinkMaxViews,
      });
      setShareLinks([...shareLinks, shareLink]);
      setShowCreateShareLink(false);
      setSelectedDomainId('');
      setShareLinkDuration('24h');
      setShareLinkPassword('');
      setShareLinkMaxViews(undefined);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create share link');
    }
  };

  const handleRevokeShareLink = async (id: string) => {
    try {
      await shareLinkApi.revoke(id);
      setShareLinks(shareLinks.map(l => l.id === id ? { ...l, isActive: false } : l));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revoke share link');
    }
  };

  const handleDeleteShareLink = async (id: string) => {
    if (!confirm('Are you sure you want to delete this share link?')) return;
    try {
      await shareLinkApi.delete(id);
      setShareLinks(shareLinks.filter(l => l.id !== id));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete share link');
    }
  };

  const handleUpdateHotReload = async (config: HotReloadConfig) => {
    try {
      const updated = await hotReloadApi.setConfig(environmentId, config);
      setHotReloadConfig(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update hot reload config');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow">
      {/* Tabs */}
      <div className="border-b border-gray-200">
        <nav className="flex -mb-px">
          {(['domains', 'share', 'hotreload'] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-6 py-3 text-sm font-medium ${
                activeTab === tab
                  ? 'border-b-2 border-blue-500 text-blue-600'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              {tab === 'domains' && 'Domains'}
              {tab === 'share' && 'Share Links'}
              {tab === 'hotreload' && 'Hot Reload'}
            </button>
          ))}
        </nav>
      </div>

      {/* Error message */}
      {error && (
        <div className="p-4 bg-red-50 border-l-4 border-red-500 text-red-700">
          {error}
          <button onClick={() => setError(null)} className="ml-4 text-red-500 hover:text-red-700">
            ×
          </button>
        </div>
      )}

      {/* Content */}
      <div className="p-6">
        {/* Domains Tab */}
        {activeTab === 'domains' && (
          <div>
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-medium">Preview Domains</h3>
              <button
                onClick={() => setShowCreateDomain(true)}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
              >
                Add Domain
              </button>
            </div>

            {/* Create Domain Form */}
            {showCreateDomain && (
              <div className="mb-6 p-4 bg-gray-50 rounded-lg">
                <h4 className="font-medium mb-4">Create New Domain</h4>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Port</label>
                    <input
                      type="number"
                      value={newDomainPort}
                      onChange={e => setNewDomainPort(parseInt(e.target.value))}
                      className="w-full px-3 py-2 border rounded-md"
                      min={1}
                      max={65535}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Type</label>
                    <select
                      value={newDomainType}
                      onChange={e => setNewDomainType(e.target.value as 'Auto' | 'Custom')}
                      className="w-full px-3 py-2 border rounded-md"
                    >
                      <option value="Auto">Auto (subdomain)</option>
                      <option value="Custom">Custom Domain</option>
                    </select>
                  </div>
                  {newDomainType === 'Custom' && (
                    <div className="col-span-2">
                      <label className="block text-sm font-medium text-gray-700 mb-1">
                        Custom Domain
                      </label>
                      <input
                        type="text"
                        value={newCustomDomain}
                        onChange={e => setNewCustomDomain(e.target.value)}
                        placeholder="myapp.example.com"
                        className="w-full px-3 py-2 border rounded-md"
                      />
                    </div>
                  )}
                </div>
                <div className="mt-4 flex gap-2">
                  <button
                    onClick={handleCreateDomain}
                    className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                  >
                    Create
                  </button>
                  <button
                    onClick={() => setShowCreateDomain(false)}
                    className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}

            {/* Domains List */}
            {domains.length === 0 ? (
              <p className="text-gray-500 text-center py-8">No domains configured</p>
            ) : (
              <div className="space-y-4">
                {domains.map(domain => (
                  <div key={domain.id} className="border rounded-lg p-4">
                    <div className="flex justify-between items-start">
                      <div>
                        <div className="flex items-center gap-2">
                          <a
                            href={`https://${domain.fullDomain}`}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-600 hover:underline font-medium"
                          >
                            {domain.fullDomain}
                          </a>
                          <span
                            className={`px-2 py-0.5 text-xs rounded-full bg-${getStatusColor(domain.status)}-100 text-${getStatusColor(domain.status)}-800`}
                          >
                            {domain.status}
                          </span>
                          {domain.sslEnabled && (
                            <span className="text-green-600 text-sm">🔒 SSL</span>
                          )}
                        </div>
                        <p className="text-sm text-gray-500 mt-1">
                          Port: {domain.port} | Type: {domain.type}
                        </p>
                      </div>
                      <div className="flex gap-2">
                        <button
                          onClick={() => copyToClipboard(`https://${domain.fullDomain}`)}
                          className="p-2 text-gray-500 hover:text-gray-700"
                          title="Copy URL"
                        >
                          📋
                        </button>
                        {domain.type === 'Custom' && !domain.dnsVerified && (
                          <button
                            onClick={() => handleVerifyDomain(domain.id)}
                            className="px-3 py-1 text-sm bg-yellow-100 text-yellow-800 rounded hover:bg-yellow-200"
                          >
                            Verify DNS
                          </button>
                        )}
                        <button
                          onClick={() => handleDeleteDomain(domain.id)}
                          className="p-2 text-red-500 hover:text-red-700"
                          title="Delete"
                        >
                          🗑️
                        </button>
                      </div>
                    </div>
                    {domain.type === 'Custom' && !domain.dnsVerified && (
                      <div className="mt-3 p-3 bg-yellow-50 rounded text-sm">
                        <p className="font-medium text-yellow-800">DNS Configuration Required</p>
                        <p className="text-yellow-700 mt-1">
                          Add a CNAME record pointing <code>{domain.customDomain}</code> to{' '}
                          <code>{domain.cnameTarget}</code>
                        </p>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Share Links Tab */}
        {activeTab === 'share' && (
          <div>
            <div className="flex justify-between items-center mb-4">
              <h3 className="text-lg font-medium">Share Links</h3>
              <button
                onClick={() => setShowCreateShareLink(true)}
                disabled={domains.filter(d => d.status === 'Active').length === 0}
                className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Create Share Link
              </button>
            </div>

            {/* Create Share Link Form */}
            {showCreateShareLink && (
              <div className="mb-6 p-4 bg-gray-50 rounded-lg">
                <h4 className="font-medium mb-4">Create Share Link</h4>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Domain</label>
                    <select
                      value={selectedDomainId}
                      onChange={e => setSelectedDomainId(e.target.value)}
                      className="w-full px-3 py-2 border rounded-md"
                    >
                      <option value="">Select a domain</option>
                      {domains
                        .filter(d => d.status === 'Active')
                        .map(d => (
                          <option key={d.id} value={d.id}>
                            {d.fullDomain}
                          </option>
                        ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">Duration</label>
                    <select
                      value={shareLinkDuration}
                      onChange={e => setShareLinkDuration(e.target.value)}
                      className="w-full px-3 py-2 border rounded-md"
                    >
                      <option value="1h">1 hour</option>
                      <option value="6h">6 hours</option>
                      <option value="12h">12 hours</option>
                      <option value="24h">24 hours</option>
                      <option value="7d">7 days</option>
                      <option value="14d">14 days</option>
                      <option value="30d">30 days</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Password (optional)
                    </label>
                    <input
                      type="password"
                      value={shareLinkPassword}
                      onChange={e => setShareLinkPassword(e.target.value)}
                      placeholder="Leave empty for no password"
                      className="w-full px-3 py-2 border rounded-md"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-1">
                      Max Views (optional)
                    </label>
                    <input
                      type="number"
                      value={shareLinkMaxViews || ''}
                      onChange={e =>
                        setShareLinkMaxViews(e.target.value ? parseInt(e.target.value) : undefined)
                      }
                      placeholder="Unlimited"
                      className="w-full px-3 py-2 border rounded-md"
                      min={1}
                    />
                  </div>
                </div>
                <div className="mt-4 flex gap-2">
                  <button
                    onClick={handleCreateShareLink}
                    disabled={!selectedDomainId}
                    className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
                  >
                    Create
                  </button>
                  <button
                    onClick={() => setShowCreateShareLink(false)}
                    className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}

            {/* Share Links List */}
            {shareLinks.length === 0 ? (
              <p className="text-gray-500 text-center py-8">No share links created</p>
            ) : (
              <div className="space-y-4">
                {shareLinks.map(link => (
                  <div key={link.id} className="border rounded-lg p-4">
                    <div className="flex justify-between items-start">
                      <div>
                        <div className="flex items-center gap-2">
                          <code className="text-sm bg-gray-100 px-2 py-1 rounded">{link.url}</code>
                          <span
                            className={`px-2 py-0.5 text-xs rounded-full ${
                              link.isActive
                                ? 'bg-green-100 text-green-800'
                                : 'bg-gray-100 text-gray-800'
                            }`}
                          >
                            {link.isActive ? 'Active' : 'Revoked'}
                          </span>
                        </div>
                        <p className="text-sm text-gray-500 mt-1">
                          {formatExpiresAt(link.expiresAt)} | Views: {link.viewCount}
                          {link.maxViews ? `/${link.maxViews}` : ''} |{' '}
                          {link.hasPassword ? '🔒 Password protected' : 'No password'}
                        </p>
                      </div>
                      <div className="flex gap-2">
                        <button
                          onClick={() => copyToClipboard(link.url)}
                          className="p-2 text-gray-500 hover:text-gray-700"
                          title="Copy URL"
                        >
                          📋
                        </button>
                        {link.isActive && (
                          <button
                            onClick={() => handleRevokeShareLink(link.id)}
                            className="px-3 py-1 text-sm bg-yellow-100 text-yellow-800 rounded hover:bg-yellow-200"
                          >
                            Revoke
                          </button>
                        )}
                        <button
                          onClick={() => handleDeleteShareLink(link.id)}
                          className="p-2 text-red-500 hover:text-red-700"
                          title="Delete"
                        >
                          🗑️
                        </button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Hot Reload Tab */}
        {activeTab === 'hotreload' && hotReloadConfig && (
          <div>
            <h3 className="text-lg font-medium mb-4">Hot Reload Configuration</h3>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <label className="font-medium">Enable Hot Reload</label>
                  <p className="text-sm text-gray-500">
                    Automatically refresh preview when files change
                  </p>
                </div>
                <button
                  onClick={() =>
                    handleUpdateHotReload({ ...hotReloadConfig, enabled: !hotReloadConfig.enabled })
                  }
                  className={`relative inline-flex h-6 w-11 items-center rounded-full ${
                    hotReloadConfig.enabled ? 'bg-blue-600' : 'bg-gray-200'
                  }`}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${
                      hotReloadConfig.enabled ? 'translate-x-6' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>

              <div>
                <label className="block font-medium mb-2">Watch Paths</label>
                <div className="flex flex-wrap gap-2">
                  {hotReloadConfig.watchPaths.map((path, i) => (
                    <span key={i} className="px-2 py-1 bg-blue-100 text-blue-800 rounded text-sm">
                      {path}
                    </span>
                  ))}
                </div>
              </div>

              <div>
                <label className="block font-medium mb-2">Ignore Paths</label>
                <div className="flex flex-wrap gap-2">
                  {hotReloadConfig.ignorePaths.map((path, i) => (
                    <span key={i} className="px-2 py-1 bg-gray-100 text-gray-800 rounded text-sm">
                      {path}
                    </span>
                  ))}
                </div>
              </div>

              <div>
                <label className="block font-medium mb-2">Debounce (ms)</label>
                <input
                  type="number"
                  value={hotReloadConfig.debounceMs}
                  onChange={e =>
                    handleUpdateHotReload({
                      ...hotReloadConfig,
                      debounceMs: parseInt(e.target.value),
                    })
                  }
                  className="w-32 px-3 py-2 border rounded-md"
                  min={0}
                  max={5000}
                />
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <label className="font-medium">Notify Clients</label>
                  <p className="text-sm text-gray-500">
                    Send refresh notifications to connected browsers
                  </p>
                </div>
                <button
                  onClick={() =>
                    handleUpdateHotReload({
                      ...hotReloadConfig,
                      notifyClients: !hotReloadConfig.notifyClients,
                    })
                  }
                  className={`relative inline-flex h-6 w-11 items-center rounded-full ${
                    hotReloadConfig.notifyClients ? 'bg-blue-600' : 'bg-gray-200'
                  }`}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition ${
                      hotReloadConfig.notifyClients ? 'translate-x-6' : 'translate-x-1'
                    }`}
                  />
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default PreviewPanel;

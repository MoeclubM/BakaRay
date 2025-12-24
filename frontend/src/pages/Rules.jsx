import { useState, useEffect } from 'react';
import { ruleAPI } from '../api';
import './Rules.css';

export default function Rules() {
  const [rules, setRules] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    protocol: 'gost',
    listen_port: '',
    mode: 'direct',
    host: '',
    port: '',
  });

  useEffect(() => {
    fetchRules();
  }, []);

  const fetchRules = async () => {
    try {
      const res = await ruleAPI.getRules();
      setRules(res.data?.data || []);
    } catch (error) {
      console.error('Failed to fetch rules:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRule = async (e) => {
    e.preventDefault();
    try {
      await ruleAPI.createRule({
        ...formData,
        targets: [{ host: formData.host, port: parseInt(formData.port) }],
      });
      setShowModal(false);
      setFormData({ name: '', protocol: 'gost', listen_port: '', mode: 'direct', host: '', port: '' });
      fetchRules();
    } catch (error) {
      alert(error.message || '创建失败');
    }
  };

  const handleDeleteRule = async (id) => {
    if (!confirm('确定要删除此规则吗？')) return;
    try {
      await ruleAPI.deleteRule(id);
      fetchRules();
    } catch (error) {
      alert(error.message || '删除失败');
    }
  };

  const formatBytes = (bytes) => {
    if (!bytes) return '0 B';
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${sizes[i]}`;
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="rules-page">
      <div className="page-header">
        <h1>转发规则</h1>
        <button className="create-btn" onClick={() => setShowModal(true)}>
          + 创建规则
        </button>
      </div>

      {rules.length > 0 ? (
        <div className="rules-table-container">
          <table className="rules-table">
            <thead>
              <tr>
                <th>名称</th>
                <th>协议</th>
                <th>监听端口</th>
                <th>模式</th>
                <th>已用流量</th>
                <th>状态</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id}>
                  <td>{rule.name}</td>
                  <td>
                    <span className={`protocol-badge ${rule.protocol}`}>
                      {rule.protocol.toUpperCase()}
                    </span>
                  </td>
                  <td>{rule.listen_port}</td>
                  <td>{rule.mode}</td>
                  <td>{formatBytes(rule.traffic_used)}</td>
                  <td>
                    <span className={`status-badge ${rule.enabled ? 'enabled' : 'disabled'}`}>
                      {rule.enabled ? '启用' : '禁用'}
                    </span>
                  </td>
                  <td>
                    <button className="delete-btn" onClick={() => handleDeleteRule(rule.id)}>
                      删除
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="empty-state">
          <p>暂无转发规则</p>
          <button className="create-btn" onClick={() => setShowModal(true)}>
            创建第一个规则
          </button>
        </div>
      )}

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>创建转发规则</h2>
            <form onSubmit={handleCreateRule}>
              <div className="form-group">
                <label>规则名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="请输入规则名称"
                  required
                />
              </div>
              <div className="form-group">
                <label>协议</label>
                <select
                  value={formData.protocol}
                  onChange={(e) => setFormData({ ...formData, protocol: e.target.value })}
                >
                  <option value="gost">Gost</option>
                  <option value="iptables">IPTables</option>
                </select>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>监听端口</label>
                  <input
                    type="number"
                    value={formData.listen_port}
                    onChange={(e) => setFormData({ ...formData, listen_port: e.target.value })}
                    placeholder="如: 8080"
                    required
                  />
                </div>
                <div className="form-group">
                  <label>模式</label>
                  <select
                    value={formData.mode}
                    onChange={(e) => setFormData({ ...formData, mode: e.target.value })}
                  >
                    <option value="direct">直连</option>
                    <option value="rr">轮询</option>
                    <option value="lb">负载均衡</option>
                  </select>
                </div>
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>目标地址</label>
                  <input
                    type="text"
                    value={formData.host}
                    onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                    placeholder="如: example.com"
                    required
                  />
                </div>
                <div className="form-group">
                  <label>目标端口</label>
                  <input
                    type="number"
                    value={formData.port}
                    onChange={(e) => setFormData({ ...formData, port: e.target.value })}
                    placeholder="如: 80"
                    required
                  />
                </div>
              </div>
              <div className="modal-actions">
                <button type="button" className="cancel-btn" onClick={() => setShowModal(false)}>
                  取消
                </button>
                <button type="submit" className="submit-btn">
                  创建
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

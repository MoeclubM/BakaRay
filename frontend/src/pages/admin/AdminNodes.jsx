import { useState, useEffect } from 'react';
import { adminNodeAPI } from '../../api';
import './Admin.css';

export default function AdminNodes() {
  const [nodes, setNodes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    host: '',
    port: '',
    secret: '',
    region: '',
    protocols: [],
    multiplier: 1,
  });

  useEffect(() => {
    fetchNodes();
  }, []);

  const fetchNodes = async () => {
    try {
      const res = await adminNodeAPI.getNodes();
      setNodes(res.data?.data || []);
    } catch (error) {
      console.error('Failed to fetch nodes:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateNode = async (e) => {
    e.preventDefault();
    try {
      await adminNodeAPI.createNode({
        ...formData,
        port: parseInt(formData.port),
        multiplier: parseFloat(formData.multiplier),
      });
      setShowModal(false);
      setFormData({ name: '', host: '', port: '', secret: '', region: '', protocols: [], multiplier: 1 });
      fetchNodes();
    } catch (error) {
      alert(error.message || '创建失败');
    }
  };

  const handleReload = async (id) => {
    try {
      await adminNodeAPI.reloadNode(id);
      alert('重载指令已发送');
    } catch (error) {
      alert(error.message || '操作失败');
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="admin-nodes">
      <div className="admin-header">
        <h2>节点管理</h2>
        <button className="create-btn" onClick={() => setShowModal(true)}>
          + 添加节点
        </button>
      </div>

      <div className="admin-table-container">
        <table className="admin-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>名称</th>
              <th>地址</th>
              <th>状态</th>
              <th>区域</th>
              <th>协议</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {nodes.map((node) => (
              <tr key={node.id}>
                <td>{node.id}</td>
                <td>{node.name}</td>
                <td>{node.host}:{node.port}</td>
                <td>
                  <span className={`status-badge ${node.status}`}>
                    {node.status === 'online' ? '在线' : '离线'}
                  </span>
                </td>
                <td>{node.region || '-'}</td>
                <td>{node.protocols?.join(', ') || '-'}</td>
                <td>
                  <button className="action-btn" onClick={() => handleReload(node.id)}>
                    重载
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>添加节点</h2>
            <form onSubmit={handleCreateNode}>
              <div className="form-group">
                <label>名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="节点名称"
                  required
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>地址</label>
                  <input
                    type="text"
                    value={formData.host}
                    onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                    placeholder="节点地址"
                    required
                  />
                </div>
                <div className="form-group">
                  <label>端口</label>
                  <input
                    type="number"
                    value={formData.port}
                    onChange={(e) => setFormData({ ...formData, port: e.target.value })}
                    placeholder="端口"
                    required
                  />
                </div>
              </div>
              <div className="form-group">
                <label>密钥</label>
                <input
                  type="text"
                  value={formData.secret}
                  onChange={(e) => setFormData({ ...formData, secret: e.target.value })}
                  placeholder="通信密钥"
                  required
                />
              </div>
              <div className="form-row">
                <div className="form-group">
                  <label>区域</label>
                  <input
                    type="text"
                    value={formData.region}
                    onChange={(e) => setFormData({ ...formData, region: e.target.value })}
                    placeholder="如: CN"
                  />
                </div>
                <div className="form-group">
                  <label>倍率</label>
                  <input
                    type="number"
                    step="0.1"
                    value={formData.multiplier}
                    onChange={(e) => setFormData({ ...formData, multiplier: e.target.value })}
                  />
                </div>
              </div>
              <div className="modal-actions">
                <button type="button" className="cancel-btn" onClick={() => setShowModal(false)}>
                  取消
                </button>
                <button type="submit" className="submit-btn">
                  添加
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

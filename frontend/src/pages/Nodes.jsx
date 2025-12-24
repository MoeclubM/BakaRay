import { useState, useEffect } from 'react';
import { nodeAPI } from '../api';
import './Nodes.css';

export default function Nodes() {
  const [nodes, setNodes] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchNodes();
  }, []);

  const fetchNodes = async () => {
    try {
      const res = await nodeAPI.getNodes();
      setNodes(res.data?.data || []);
    } catch (error) {
      console.error('Failed to fetch nodes:', error);
    } finally {
      setLoading(false);
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
    <div className="nodes-page">
      <h1>节点列表</h1>

      {nodes.length > 0 ? (
        <div className="nodes-grid">
          {nodes.map((node) => (
            <div key={node.id} className={`node-card ${node.status}`}>
              <div className="node-header">
                <h3>{node.name}</h3>
                <span className={`status-badge ${node.status}`}>
                  {node.status === 'online' ? '在线' : '离线'}
                </span>
              </div>
              <div className="node-details">
                <div className="detail-item">
                  <span className="label">地址</span>
                  <span className="value">{node.host}:{node.port}</span>
                </div>
                <div className="detail-item">
                  <span className="label">区域</span>
                  <span className="value">{node.region || '未知'}</span>
                </div>
                <div className="detail-item">
                  <span className="label">协议</span>
                  <span className="value">{node.protocols?.join(', ') || 'N/A'}</span>
                </div>
                <div className="detail-item">
                  <span className="label">倍率</span>
                  <span className="value">{node.multiplier}x</span>
                </div>
              </div>
              {node.last_seen && (
                <div className="node-footer">
                  最后活跃: {new Date(node.last_seen).toLocaleString('zh-CN')}
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="empty-state">
          <p>暂无节点信息</p>
        </div>
      )}
    </div>
  );
}

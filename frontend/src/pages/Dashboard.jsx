import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import { userAPI, nodeAPI } from '../api';
import './Dashboard.css';

export default function Dashboard() {
  const { user } = useAuth();
  const [profile, setProfile] = useState(null);
  const [trafficStats, setTrafficStats] = useState(null);
  const [nodes, setNodes] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      const [profileRes, trafficRes, nodesRes] = await Promise.all([
        userAPI.getProfile(),
        userAPI.getTrafficStats({ days: 7 }),
        nodeAPI.getNodes().catch(() => ({ data: { data: [] } })),
      ]);
      setProfile(profileRes.data);
      setTrafficStats(trafficRes.data);
      setNodes(nodesRes.data?.data || []);
    } catch (error) {
      console.error('Failed to fetch data:', error);
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

  const formatCurrency = (cents) => {
    return `¥${(cents / 100).toFixed(2)}`;
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  const onlineNodes = nodes.filter(n => n.status === 'online').length;

  return (
    <div className="dashboard">
      <h1>欢迎回来，{user?.username}</h1>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-icon">💰</div>
          <div className="stat-content">
            <span className="stat-label">账户余额</span>
            <span className="stat-value">{formatCurrency(profile?.balance || 0)}</span>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">📊</div>
          <div className="stat-content">
            <span className="stat-label">已用流量</span>
            <span className="stat-value">{formatBytes(trafficStats?.totalUsed || 0)}</span>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">🖥️</div>
          <div className="stat-content">
            <span className="stat-label">可用节点</span>
            <span className="stat-value">{onlineNodes} / {nodes.length}</span>
          </div>
        </div>

        <div className="stat-card">
          <div className="stat-icon">📅</div>
          <div className="stat-content">
            <span className="stat-label">注册时间</span>
            <span className="stat-value">
              {profile?.created_at ? new Date(profile.created_at).toLocaleDateString('zh-CN') : '-'}
            </span>
          </div>
        </div>
      </div>

      <div className="dashboard-section">
        <h2>节点状态</h2>
        {nodes.length > 0 ? (
          <div className="node-list">
            {nodes.map((node) => (
              <div key={node.id} className="node-item">
                <div className="node-info">
                  <span className="node-name">{node.name}</span>
                  <span className="node-host">{node.host}:{node.port}</span>
                </div>
                <div className={`node-status ${node.status}`}>
                  {node.status === 'online' ? '在线' : '离线'}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="empty-state">暂无节点</div>
        )}
      </div>

      <div className="dashboard-section">
        <h2>账户信息</h2>
        <div className="profile-info">
          <div className="info-item">
            <span className="info-label">用户名</span>
            <span className="info-value">{profile?.username}</span>
          </div>
          <div className="info-item">
            <span className="info-label">用户组</span>
            <span className="info-value">{profile?.user_group_id ? `组#${profile.user_group_id}` : '默认组'}</span>
          </div>
          <div className="info-item">
            <span className="info-label">角色</span>
            <span className="info-value">{profile?.role === 'admin' ? '管理员' : '普通用户'}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

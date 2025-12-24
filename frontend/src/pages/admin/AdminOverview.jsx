import { useState, useEffect } from 'react';
import { adminNodeAPI, adminUserAPI, adminOrderAPI } from '../../api';
import './Admin.css';

export default function AdminOverview() {
  const [stats, setStats] = useState({
    nodes: { total: 0, online: 0 },
    users: { total: 0 },
    orders: { total: 0, pending: 0 },
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchStats();
  }, []);

  const fetchStats = async () => {
    try {
      const [nodesRes, usersRes, ordersRes] = await Promise.all([
        adminNodeAPI.getNodes().catch(() => ({ data: { data: [] } })),
        adminUserAPI.getUsers().catch(() => ({ data: { data: [] } })),
        adminOrderAPI.getOrders().catch(() => ({ data: { data: [] } })),
      ]);

      const nodes = nodesRes.data?.data || [];
      const users = usersRes.data?.data || [];
      const orders = ordersRes.data?.data || [];

      setStats({
        nodes: {
          total: nodes.length,
          online: nodes.filter((n) => n.status === 'online').length,
        },
        users: { total: users.length },
        orders: {
          total: orders.length,
          pending: orders.filter((o) => o.status === 'pending').length,
        },
      });
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="admin-overview">
      <div className="overview-grid">
        <div className="overview-card">
          <div className="card-icon">🖥️</div>
          <div className="card-content">
            <span className="card-value">{stats.nodes.total}</span>
            <span className="card-label">节点总数</span>
            <span className="card-sub">{stats.nodes.online} 个在线</span>
          </div>
        </div>

        <div className="overview-card">
          <div className="card-icon">👥</div>
          <div className="card-content">
            <span className="card-value">{stats.users.total}</span>
            <span className="card-label">用户总数</span>
          </div>
        </div>

        <div className="overview-card">
          <div className="card-icon">📋</div>
          <div className="card-content">
            <span className="card-value">{stats.orders.total}</span>
            <span className="card-label">订单总数</span>
            <span className="card-sub">{stats.orders.pending} 个待处理</span>
          </div>
        </div>
      </div>
    </div>
  );
}

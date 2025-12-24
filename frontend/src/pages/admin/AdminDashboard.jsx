import { useState, useEffect } from 'react';
import { Link, Outlet } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';
import './Admin.css';

export default function AdminDashboard() {
  const { isAdmin } = useAuth();
  const [activeTab, setActiveTab] = useState('overview');

  if (!isAdmin) {
    return (
      <div className="admin-error">
        <h2>权限不足</h2>
        <p>您没有访问管理后台的权限</p>
      </div>
    );
  }

  const tabs = [
    { id: 'overview', label: '概览', path: '/admin' },
    { id: 'nodes', label: '节点管理', path: '/admin/nodes' },
    { id: 'users', label: '用户管理', path: '/admin/users' },
    { id: 'packages', label: '套餐管理', path: '/admin/packages' },
    { id: 'orders', label: '订单管理', path: '/admin/orders' },
  ];

  return (
    <div className="admin-page">
      <h1>管理后台</h1>
      <div className="admin-tabs">
        {tabs.map((tab) => (
          <Link
            key={tab.id}
            to={tab.path}
            className={`admin-tab ${activeTab === tab.id ? 'active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </Link>
        ))}
      </div>
      <div className="admin-content">
        <Outlet />
      </div>
    </div>
  );
}

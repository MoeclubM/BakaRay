import { useState } from 'react';
import { Outlet, Link, useNavigate, useLocation } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import './Layout.css';

export default function Layout() {
  const { user, logout, isAdmin } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const menuItems = [
    { path: '/dashboard', label: '仪表盘', icon: '📊' },
    { path: '/nodes', label: '节点列表', icon: '🖥️' },
    { path: '/rules', label: '规则管理', icon: '🔀' },
    { path: '/packages', label: '套餐购买', icon: '📦' },
    { path: '/orders', label: '我的订单', icon: '📋' },
  ];

  const adminMenuItems = [
    { path: '/admin', label: '管理后台', icon: '⚙️' },
    { path: '/admin/nodes', label: '节点管理', icon: '🖥️' },
    { path: '/admin/users', label: '用户管理', icon: '👥' },
    { path: '/admin/packages', label: '套餐管理', icon: '📦' },
    { path: '/admin/orders', label: '订单管理', icon: '📋' },
  ];

  return (
    <div className="layout">
      <aside className={`sidebar ${sidebarCollapsed ? 'collapsed' : ''}`}>
        <div className="sidebar-header">
          <h2>BakaRay</h2>
          <button
            className="collapse-btn"
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
          >
            {sidebarCollapsed ? '→' : '←'}
          </button>
        </div>
        <nav className="sidebar-nav">
          {!isAdmin && menuItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              className={`nav-item ${location.pathname === item.path ? 'active' : ''}`}
            >
              <span className="nav-icon">{item.icon}</span>
              {!sidebarCollapsed && <span className="nav-label">{item.label}</span>}
            </Link>
          ))}
          {isAdmin && adminMenuItems.map((item) => (
            <Link
              key={item.path}
              to={item.path}
              className={`nav-item ${location.pathname === item.path ? 'active' : ''}`}
            >
              <span className="nav-icon">{item.icon}</span>
              {!sidebarCollapsed && <span className="nav-label">{item.label}</span>}
            </Link>
          ))}
        </nav>
        <div className="sidebar-footer">
          <div className="user-info">
            <span className="user-avatar">👤</span>
            {!sidebarCollapsed && (
              <div className="user-details">
                <span className="user-name">{user?.username}</span>
                <span className="user-role">{user?.role === 'admin' ? '管理员' : '用户'}</span>
              </div>
            )}
          </div>
          <button className="logout-btn" onClick={handleLogout}>
            {sidebarCollapsed ? '🚪' : '退出登录'}
          </button>
        </div>
      </aside>
      <main className="main-content">
        <Outlet />
      </main>
    </div>
  );
}

import { useState, useEffect } from 'react';
import { adminUserAPI } from '../../api';
import './Admin.css';

export default function AdminUsers() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [balanceModal, setBalanceModal] = useState(false);
  const [selectedUser, setSelectedUser] = useState(null);
  const [formData, setFormData] = useState({
    username: '',
    password: '',
    role: 'user',
  });
  const [balanceData, setBalanceData] = useState({ amount: 0, type: 'add' });

  useEffect(() => {
    fetchUsers();
  }, []);

  const fetchUsers = async () => {
    try {
      const res = await adminUserAPI.getUsers();
      setUsers(res.data?.data || []);
    } catch (error) {
      console.error('Failed to fetch users:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateUser = async (e) => {
    e.preventDefault();
    try {
      await adminUserAPI.createUser(formData);
      setShowModal(false);
      setFormData({ username: '', password: '', role: 'user' });
      fetchUsers();
    } catch (error) {
      alert(error.message || '创建失败');
    }
  };

  const handleAdjustBalance = async (e) => {
    e.preventDefault();
    try {
      await adminUserAPI.adjustBalance(selectedUser.id, {
        amount: balanceData.type === 'add' ? Math.abs(balanceData.amount) : -Math.abs(balanceData.amount),
      });
      setBalanceModal(false);
      setSelectedUser(null);
      setBalanceData({ amount: 0, type: 'add' });
      fetchUsers();
    } catch (error) {
      alert(error.message || '操作失败');
    }
  };

  const formatCurrency = (cents) => {
    return `¥${(cents / 100).toFixed(2)}`;
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="admin-users">
      <div className="admin-header">
        <h2>用户管理</h2>
        <button className="create-btn" onClick={() => setShowModal(true)}>
          + 添加用户
        </button>
      </div>

      <div className="admin-table-container">
        <table className="admin-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>用户名</th>
              <th>余额</th>
              <th>角色</th>
              <th>注册时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id}>
                <td>{user.id}</td>
                <td>{user.username}</td>
                <td className="amount">{formatCurrency(user.balance || 0)}</td>
                <td>
                  <span className={`role-badge ${user.role}`}>
                    {user.role === 'admin' ? '管理员' : '用户'}
                  </span>
                </td>
                <td>{new Date(user.created_at).toLocaleDateString('zh-CN')}</td>
                <td>
                  <button
                    className="action-btn"
                    onClick={() => {
                      setSelectedUser(user);
                      setBalanceModal(true);
                    }}
                  >
                    调整余额
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
            <h2>添加用户</h2>
            <form onSubmit={handleCreateUser}>
              <div className="form-group">
                <label>用户名</label>
                <input
                  type="text"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  placeholder="用户名"
                  required
                />
              </div>
              <div className="form-group">
                <label>密码</label>
                <input
                  type="password"
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  placeholder="密码"
                  required
                />
              </div>
              <div className="form-group">
                <label>角色</label>
                <select
                  value={formData.role}
                  onChange={(e) => setFormData({ ...formData, role: e.target.value })}
                >
                  <option value="user">普通用户</option>
                  <option value="admin">管理员</option>
                </select>
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

      {balanceModal && (
        <div className="modal-overlay" onClick={() => setBalanceModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>调整余额 - {selectedUser?.username}</h2>
            <form onSubmit={handleAdjustBalance}>
              <div className="form-group">
                <label>操作类型</label>
                <select
                  value={balanceData.type}
                  onChange={(e) => setBalanceData({ ...balanceData, type: e.target.value })}
                >
                  <option value="add">增加余额</option>
                  <option value="sub">减少余额</option>
                </select>
              </div>
              <div className="form-group">
                <label>金额（分）</label>
                <input
                  type="number"
                  value={balanceData.amount}
                  onChange={(e) => setBalanceData({ ...balanceData, amount: e.target.value })}
                  placeholder="请输入金额（分）"
                  required
                />
              </div>
              <div className="modal-actions">
                <button type="button" className="cancel-btn" onClick={() => setBalanceModal(false)}>
                  取消
                </button>
                <button type="submit" className="submit-btn">
                  确认
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

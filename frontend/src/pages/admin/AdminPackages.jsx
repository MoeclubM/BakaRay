import { useState, useEffect } from 'react';
import { adminPackageAPI } from '../../api';
import './Admin.css';

export default function AdminPackages() {
  const [packages, setPackages] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [formData, setFormData] = useState({
    name: '',
    traffic: '',
    price: '',
    user_group_id: '',
  });

  useEffect(() => {
    fetchPackages();
  }, []);

  const fetchPackages = async () => {
    try {
      const res = await adminPackageAPI.getPackages();
      setPackages(res.data?.data || []);
    } catch (error) {
      console.error('Failed to fetch packages:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreatePackage = async (e) => {
    e.preventDefault();
    try {
      await adminPackageAPI.createPackage({
        ...formData,
        traffic: parseInt(formData.traffic) * 1024 * 1024 * 1024,
        price: parseInt(formData.price) * 100,
        user_group_id: formData.user_group_id ? parseInt(formData.user_group_id) : null,
      });
      setShowModal(false);
      setFormData({ name: '', traffic: '', price: '', user_group_id: '' });
      fetchPackages();
    } catch (error) {
      alert(error.message || '创建失败');
    }
  };

  const formatBytes = (gb) => {
    if (!gb) return '0 GB';
    const tb = gb / 1024;
    return tb >= 1 ? `${tb.toFixed(0)} TB` : `${gb} GB`;
  };

  const formatCurrency = (cents) => {
    return `¥${(cents / 100).toFixed(2)}`;
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="admin-packages">
      <div className="admin-header">
        <h2>套餐管理</h2>
        <button className="create-btn" onClick={() => setShowModal(true)}>
          + 添加套餐
        </button>
      </div>

      <div className="admin-table-container">
        <table className="admin-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>名称</th>
              <th>流量</th>
              <th>价格</th>
              <th>用户组</th>
            </tr>
          </thead>
          <tbody>
            {packages.map((pkg) => (
              <tr key={pkg.id}>
                <td>{pkg.id}</td>
                <td>{pkg.name}</td>
                <td>{formatBytes(pkg.traffic / (1024 * 1024 * 1024))}</td>
                <td className="amount">{formatCurrency(pkg.price)}</td>
                <td>{pkg.user_group_id ? `组#${pkg.user_group_id}` : '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showModal && (
        <div className="modal-overlay" onClick={() => setShowModal(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h2>添加套餐</h2>
            <form onSubmit={handleCreatePackage}>
              <div className="form-group">
                <label>套餐名称</label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="如: 月租套餐"
                  required
                />
              </div>
              <div className="form-group">
                <label>流量（GB）</label>
                <input
                  type="number"
                  value={formData.traffic}
                  onChange={(e) => setFormData({ ...formData, traffic: e.target.value })}
                  placeholder="如: 100"
                  required
                />
              </div>
              <div className="form-group">
                <label>价格（元）</label>
                <input
                  type="number"
                  value={formData.price}
                  onChange={(e) => setFormData({ ...formData, price: e.target.value })}
                  placeholder="如: 29.9"
                  step="0.01"
                  required
                />
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

import { useState, useEffect } from 'react';
import { packageAPI, orderAPI, depositAPI } from '../api';
import './Packages.css';

export default function Packages() {
  const [packages, setPackages] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchPackages();
  }, []);

  const fetchPackages = async () => {
    try {
      const res = await packageAPI.getPackages();
      setPackages(res.data?.data || []);
    } catch (error) {
      console.error('Failed to fetch packages:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatBytes = (bytes) => {
    if (!bytes) return '0 B';
    const sizes = ['GB', 'TB'];
    const i = bytes >= 1024 * 1024 * 1024 ? 1 : 0;
    const value = bytes / Math.pow(1024, i + (i === 0 ? 2 : 3));
    return `${value.toFixed(0)} ${sizes[i]}`;
  };

  const formatCurrency = (cents) => {
    return `¥${(cents / 100).toFixed(2)}`;
  };

  const handlePurchase = async (pkg) => {
    try {
      // 创建订单
      const orderRes = await orderAPI.createOrder({ package_id: pkg.id });
      const orderId = orderRes.data?.id;
      if (!orderId) {
        alert('订单创建失败');
        return;
      }
      // 发起充值
      const depositRes = await depositAPI.deposit({
        order_id: orderId,
        amount: pkg.price,
        pay_type: 'test',
      });
      alert(`订单创建成功！订单号: ${orderId}\n注意: 这是一个测试充值`);
    } catch (error) {
      alert(error.message || '购买失败');
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="packages-page">
      <h1>套餐购买</h1>

      {packages.length > 0 ? (
        <div className="packages-grid">
          {packages.map((pkg) => (
            <div key={pkg.id} className="package-card">
              <div className="package-header">
                <h3>{pkg.name}</h3>
              </div>
              <div className="package-body">
                <div className="package-traffic">
                  <span className="traffic-icon">📊</span>
                  <span className="traffic-value">{formatBytes(pkg.traffic)}</span>
                  <span className="traffic-label">流量</span>
                </div>
                <div className="package-price">
                  <span className="price-value">{formatCurrency(pkg.price)}</span>
                  <span className="price-label">/ 月</span>
                </div>
              </div>
              <div className="package-footer">
                <button className="purchase-btn" onClick={() => handlePurchase(pkg)}>
                  立即购买
                </button>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="empty-state">
          <p>暂无套餐</p>
        </div>
      )}
    </div>
  );
}

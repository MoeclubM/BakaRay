import { useState, useEffect } from 'react';
import { adminOrderAPI } from '../../api';
import './Admin.css';

export default function AdminOrders() {
  const [orders, setOrders] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchOrders();
  }, []);

  const fetchOrders = async () => {
    try {
      const res = await adminOrderAPI.getOrders();
      setOrders(res.data?.data || []);
    } catch (error) {
      console.error('Failed to fetch orders:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatCurrency = (cents) => {
    return `¥${(cents / 100).toFixed(2)}`;
  };

  const formatDate = (date) => {
    return new Date(date).toLocaleString('zh-CN');
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="admin-orders">
      <div className="admin-header">
        <h2>订单管理</h2>
      </div>

      <div className="admin-table-container">
        <table className="admin-table">
          <thead>
            <tr>
              <th>ID</th>
              <th>订单号</th>
              <th>金额</th>
              <th>支付方式</th>
              <th>状态</th>
              <th>创建时间</th>
            </tr>
          </thead>
          <tbody>
            {orders.map((order) => (
              <tr key={order.id}>
                <td>{order.id}</td>
                <td className="trade-no">{order.trade_no}</td>
                <td className="amount">{formatCurrency(order.amount)}</td>
                <td>{order.pay_type || '-'}</td>
                <td>
                  <span className={`status-badge ${order.status}`}>
                    {order.status === 'pending' ? '待支付' : order.status === 'success' ? '成功' : '失败'}
                  </span>
                </td>
                <td className="time">{formatDate(order.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

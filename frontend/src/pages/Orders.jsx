import { useState, useEffect } from 'react';
import { orderAPI } from '../api';
import './Orders.css';

export default function Orders() {
  const [orders, setOrders] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchOrders();
  }, []);

  const fetchOrders = async () => {
    try {
      const res = await orderAPI.getOrders();
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
    <div className="orders-page">
      <h1>我的订单</h1>

      {orders.length > 0 ? (
        <div className="orders-table-container">
          <table className="orders-table">
            <thead>
              <tr>
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
      ) : (
        <div className="empty-state">
          <p>暂无订单记录</p>
        </div>
      )}
    </div>
  );
}

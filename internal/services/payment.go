package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"bakaray/internal/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var ErrPackageNotFound = errors.New("套餐不存在")
var ErrOrderNotFound = errors.New("订单不存在")

// PaymentService 支付服务
type PaymentService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewPaymentService 创建支付服务
func NewPaymentService(db *gorm.DB, redis *redis.Client) *PaymentService {
	return &PaymentService{db: db, redis: redis}
}

// GetUserByID 获取用户（用于支付服务）
func (s *PaymentService) GetUserByID(userID uint) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// CreatePackage 创建套餐
func (s *PaymentService) CreatePackage(pkg *models.Package) error {
	return s.db.Create(pkg).Error
}

// GetPackageByID 获取套餐
func (s *PaymentService) GetPackageByID(id uint) (*models.Package, error) {
	var pkg models.Package
	if err := s.db.First(&pkg, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPackageNotFound
		}
		return nil, err
	}
	return &pkg, nil
}

// ListPackages 获取套餐列表
func (s *PaymentService) ListPackages(userGroupID uint) ([]models.Package, error) {
	var packages []models.Package
	query := s.db
	if userGroupID > 0 {
		query = query.Where("user_group_id = ? OR user_group_id = 0", userGroupID)
	}
	if err := query.Find(&packages).Error; err != nil {
		return nil, err
	}
	return packages, nil
}

// ListVisiblePackages 获取可见的套餐列表
func (s *PaymentService) ListVisiblePackages(userGroupID uint) ([]models.Package, error) {
	var packages []models.Package
	query := s.db.Where("visible = ?", true)
	if userGroupID > 0 {
		query = query.Where("user_group_id = ? OR user_group_id = 0", userGroupID)
	}
	if err := query.Find(&packages).Error; err != nil {
		return nil, err
	}
	return packages, nil
}

// HasUserPurchasedPackage 检查用户是否已购买过该套餐
func (s *PaymentService) HasUserPurchasedPackage(userID, packageID uint) (bool, error) {
	var count int64
	err := s.db.Model(&models.Order{}).
		Where("user_id = ? AND package_id = ? AND status = ?", userID, packageID, "success").
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateOrder 创建订单
func (s *PaymentService) CreateOrder(userID, packageID uint, amount int64, payType string) (*models.Order, error) {
	order := &models.Order{
		UserID:    userID,
		PackageID: packageID,
		Amount:    amount,
		Status:    "pending",
		TradeNo:   generateTradeNo(),
		PayType:   payType,
	}

	if err := s.db.Create(order).Error; err != nil {
		return nil, err
	}
	return order, nil
}

// CreateAndCompleteOrder 创建并完成订单（余额支付）
func (s *PaymentService) CreateAndCompleteOrder(userID, packageID uint, amount int64) (*models.Order, error) {
	// 获取套餐信息
	pkg, err := s.GetPackageByID(packageID)
	if err != nil {
		return nil, err
	}

	// 在事务中完成订单和扣款
	var order *models.Order
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// 检查用户余额是否足够
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if user.Balance < amount {
			return errors.New("余额不足")
		}

		// 创建已完成的订单
		order = &models.Order{
			UserID:    userID,
			PackageID: packageID,
			Amount:    amount,
			Status:    "success",
			TradeNo:   generateTradeNo(),
			PayType:   "balance",
		}
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// 扣款（从余额中扣除金额）
		if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("balance", gorm.Expr("balance - ?", amount)).Error; err != nil {
			return err
		}

		// 增加用户流量
		if pkg.Traffic > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("traffic_balance", gorm.Expr("traffic_balance + ?", pkg.Traffic)).Error; err != nil {
				return err
			}
		}

		// 如果套餐指定了用户组，更新用户的用户组
		if pkg.UserGroupID > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("user_group_id", pkg.UserGroupID).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return order, nil
}

// GetOrderByTradeNo 根据交易号获取订单
func (s *PaymentService) GetOrderByTradeNo(tradeNo string) (*models.Order, error) {
	var order models.Order
	if err := s.db.Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

// GetOrderByID 根据ID获取订单
func (s *PaymentService) GetOrderByID(id uint) (*models.Order, error) {
	var order models.Order
	if err := s.db.First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

// AddUserBalance 增加用户余额
func (s *PaymentService) AddUserBalance(userID uint, amount int64) error {
	return s.db.Model(&models.User{}).Where("id = ?", userID).Update("balance", gorm.Expr("balance + ?", amount)).Error
}

// UpdateOrderStatus 更新订单状态
func (s *PaymentService) UpdateOrderStatus(tradeNo string, status string) error {
	return s.db.Model(&models.Order{}).Where("trade_no = ?", tradeNo).Update("status", status).Error
}

// CompleteOrder 完成订单（支付成功）
func (s *PaymentService) CompleteOrder(tradeNo string, userID uint, traffic int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新订单状态
		if err := tx.Model(&models.Order{}).Where("trade_no = ?", tradeNo).Update("status", "success").Error; err != nil {
			return err
		}
		// 更新用户流量
		if traffic > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("traffic_balance", gorm.Expr("traffic_balance + ?", traffic)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// CompleteOrderWithLock 幂等性完成订单（带分布式锁）
func (s *PaymentService) CompleteOrderWithLock(tradeNo string, userID uint, traffic int64) error {
	// 如果没有 Redis，使用普通方法
	if s.redis == nil {
		return s.CompleteOrder(tradeNo, userID, traffic)
	}

	ctx := context.Background()
	lockKey := "order:lock:" + tradeNo

	// 获取分布式锁
	lockAcquired, err := s.redis.SetNX(ctx, lockKey, "1", 10*time.Second).Result()
	if err != nil {
		// 如果 Redis 错误，回退到普通方法
		return s.CompleteOrder(tradeNo, userID, traffic)
	}

	if !lockAcquired {
		// 锁已被持有，返回成功（幂等性）
		return nil
	}
	defer s.redis.Del(ctx, lockKey)

	// 检查订单是否已完成（幂等性检查）
	var order models.Order
	if err := s.db.Where("trade_no = ?", tradeNo).First(&order).Error; err != nil {
		return err
	}
	if order.Status == "success" {
		// 订单已完成，直接返回
		return nil
	}

	// 获取套餐信息
	pkg, err := s.GetPackageByID(order.PackageID)
	if err != nil {
		return err
	}

	// 在事务中完成订单
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 更新订单状态
		if err := tx.Model(&models.Order{}).Where("trade_no = ?", tradeNo).Update("status", "success").Error; err != nil {
			return err
		}
		// 更新用户流量
		if pkg.Traffic > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("traffic_balance", gorm.Expr("traffic_balance + ?", pkg.Traffic)).Error; err != nil {
				return err
			}
		}
		// 如果套餐指定了用户组，更新用户的用户组
		if pkg.UserGroupID > 0 {
			if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("user_group_id", pkg.UserGroupID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListOrders 获取订单列表
func (s *PaymentService) ListOrders(userID uint, page, pageSize int) ([]models.Order, int64) {
	var orders []models.Order
	var total int64

	query := s.db.Model(&models.Order{})
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders)

	return orders, total
}

// ListAllOrders 获取所有订单（管理员用）
func (s *PaymentService) ListAllOrders(page, pageSize int, status string) ([]models.Order, int64) {
	var orders []models.Order
	var total int64

	query := s.db.Model(&models.Order{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders)

	return orders, total
}

// GetOrderStats 获取订单统计数据
func (s *PaymentService) GetOrderStats() (int64, int64) {
	var orderCount int64
	var totalRevenue int64

	s.db.Model(&models.Order{}).Where("status = ?", "success").Count(&orderCount)
	s.db.Model(&models.Order{}).Where("status = ?", "success").Select("COALESCE(SUM(amount), 0)").Scan(&totalRevenue)

	return orderCount, totalRevenue
}

// PaymentConfigService 支付配置服务
type PaymentConfigService struct {
	db *gorm.DB
}

// NewPaymentConfigService 创建支付配置服务
func NewPaymentConfigService(db *gorm.DB) *PaymentConfigService {
	return &PaymentConfigService{db: db}
}

// CreatePaymentConfig 创建支付配置
func (s *PaymentConfigService) CreatePaymentConfig(cfg *models.PaymentConfig) error {
	return s.db.Create(cfg).Error
}

// GetPaymentConfig 获取支付配置
func (s *PaymentConfigService) GetPaymentConfig(id uint) (*models.PaymentConfig, error) {
	var cfg models.PaymentConfig
	if err := s.db.First(&cfg, id).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ListPaymentConfigs 获取支付配置列表
func (s *PaymentConfigService) ListPaymentConfigs() ([]models.PaymentConfig, error) {
	var configs []models.PaymentConfig
	if err := s.db.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetPaymentConfigByType 根据类型获取支付配置
func (s *PaymentConfigService) GetPaymentConfigByType(payType string) (*models.PaymentConfig, error) {
	var cfg models.PaymentConfig
	// 使用ID或名称查找
	if err := s.db.Where("id = ? OR name = ?", payType, payType).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetPaymentConfigForCallback tries to locate the payment config using callback parameters.
func (s *PaymentConfigService) GetPaymentConfigForCallback(params map[string]string) (*models.PaymentConfig, error) {
	// Preferred: explicit pay_type configured by caller.
	if payType := params["pay_type"]; payType != "" {
		return s.GetPaymentConfigByType(payType)
	}

	// Epay includes merchant id.
	if pid := params["pid"]; pid != "" {
		var cfg models.PaymentConfig
		if err := s.db.Where("merchant_id = ? AND enabled = ?", pid, true).First(&cfg).Error; err == nil {
			return &cfg, nil
		}
	}

	// Fallback to first enabled config.
	var cfg models.PaymentConfig
	if err := s.db.Where("enabled = ?", true).Order("id ASC").First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdatePaymentConfig 更新支付配置
func (s *PaymentConfigService) UpdatePaymentConfig(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.PaymentConfig{}).Where("id = ?", id).Updates(updates).Error
}

// DeletePaymentConfig 删除支付配置
func (s *PaymentConfigService) DeletePaymentConfig(id uint) error {
	return s.db.Delete(&models.PaymentConfig{}, id).Error
}

// UpdatePackage 更新套餐
func (s *PaymentService) UpdatePackage(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.Package{}).Where("id = ?", id).Updates(updates).Error
}

// DeletePackage 删除套餐
func (s *PaymentService) DeletePackage(id uint) error {
	return s.db.Delete(&models.Package{}, id).Error
}

// generateTradeNo 生成交易号
func generateTradeNo() string {
	now := time.Now().Format("20060102150405")
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return now
	}
	return now + "-" + hex.EncodeToString(buf)
}

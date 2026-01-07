package services

import (
	"testing"
	"time"

	"bakaray/internal/models"

	"github.com/stretchr/testify/require"
)

func TestCreateOrder(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建测试用户
	user := &models.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Balance:      0,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(user).Error)

	// 创建测试套餐
	pkg := &models.Package{
		Name:    "测试套餐",
		Traffic: 1024 * 1024 * 1024, // 1GB
		Price:   1000,               // 10元
	}
	require.NoError(t, db.Create(pkg).Error)

	svc := NewPaymentService(db, nil)

	// 测试成功创建订单
	order, err := svc.CreateOrder(user.ID, pkg.ID, 1000, "alipay")
	require.NoError(t, err)
	require.NotNil(t, order)
	require.Equal(t, user.ID, order.UserID)
	require.Equal(t, pkg.ID, order.PackageID)
	require.Equal(t, int64(1000), order.Amount)
	require.Equal(t, "pending", order.Status)
	require.NotEmpty(t, order.TradeNo)

	// 测试订单号生成唯一性
	order2, err := svc.CreateOrder(user.ID, pkg.ID, 1000, "wechat")
	require.NoError(t, err)
	require.NotEqual(t, order.TradeNo, order2.TradeNo)
}

func TestGetOrderByTradeNo(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建测试用户
	user := &models.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Balance:      0,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(user).Error)

	svc := NewPaymentService(db, nil)

	// 创建测试订单
	tradeNo := generateTradeNo()
	order := &models.Order{
		UserID:  user.ID,
		Amount:  1000,
		Status:  "pending",
		TradeNo: tradeNo,
		PayType: "alipay",
	}
	require.NoError(t, db.Create(order).Error)

	// 测试成功获取订单
	foundOrder, err := svc.GetOrderByTradeNo(tradeNo)
	require.NoError(t, err)
	require.NotNil(t, foundOrder)
	require.Equal(t, tradeNo, foundOrder.TradeNo)
	require.Equal(t, user.ID, foundOrder.UserID)

	// 测试订单不存在时返回错误
	_, err = svc.GetOrderByTradeNo("non_existent_trade_no")
	require.Error(t, err)
	require.Equal(t, ErrOrderNotFound, err)
}

func TestListOrders(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建测试用户
	user := &models.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Balance:      0,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(user).Error)

	svc := NewPaymentService(db, nil)

	// 创建多个订单
	for i := 0; i < 5; i++ {
		order := &models.Order{
			UserID:  user.ID,
			Amount:  int64(1000 * (i + 1)),
			Status:  "pending",
			TradeNo: generateTradeNo(),
			PayType: "alipay",
		}
		require.NoError(t, db.Create(order).Error)
	}

	// 创建另一个用户的订单
	otherUser := &models.User{
		Username:     "otheruser",
		PasswordHash: "hashedpassword",
		Balance:      0,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(otherUser).Error)
	otherOrder := &models.Order{
		UserID:  otherUser.ID,
		Amount:  2000,
		Status:  "pending",
		TradeNo: generateTradeNo(),
		PayType: "wechat",
	}
	require.NoError(t, db.Create(otherOrder).Error)

	// 测试获取用户订单
	orders, total := svc.ListOrders(user.ID, 1, 10)
	require.Len(t, orders, 5)
	require.Equal(t, int64(5), total)

	// 测试分页
	orders, total = svc.ListOrders(user.ID, 1, 2)
	require.Len(t, orders, 2)
	require.Equal(t, int64(5), total)

	// 测试第二页
	orders, total = svc.ListOrders(user.ID, 2, 2)
	require.Len(t, orders, 2)
	require.Equal(t, int64(5), total)

	// 测试分页超出范围
	orders, total = svc.ListOrders(user.ID, 10, 10)
	require.Len(t, orders, 0)
	require.Equal(t, int64(5), total)
}

func TestListAllOrders(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建测试用户
	user := &models.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Balance:      0,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(user).Error)

	svc := NewPaymentService(db, nil)

	// 创建不同状态的订单
	orderStatuses := []string{"pending", "success", "failed", "success", "pending"}
	for i, status := range orderStatuses {
		order := &models.Order{
			UserID:  user.ID,
			Amount:  int64(1000 * (i + 1)),
			Status:  status,
			TradeNo: generateTradeNo(),
			PayType: "alipay",
		}
		require.NoError(t, db.Create(order).Error)
	}

	// 测试获取所有订单
	orders, total := svc.ListAllOrders(1, 10, "")
	require.Len(t, orders, 5)
	require.Equal(t, int64(5), total)

	// 测试按状态筛选 - pending
	orders, total = svc.ListAllOrders(1, 10, "pending")
	require.Len(t, orders, 2)
	require.Equal(t, int64(2), total)
	for _, o := range orders {
		require.Equal(t, "pending", o.Status)
	}

	// 测试按状态筛选 - success
	orders, total = svc.ListAllOrders(1, 10, "success")
	require.Len(t, orders, 2)
	require.Equal(t, int64(2), total)
	for _, o := range orders {
		require.Equal(t, "success", o.Status)
	}

	// 测试按状态筛选 - failed
	orders, total = svc.ListAllOrders(1, 10, "failed")
	require.Len(t, orders, 1)
	require.Equal(t, int64(1), total)
	require.Equal(t, "failed", orders[0].Status)
}

func TestUpdateOrderStatus(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建测试用户
	user := &models.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Balance:      0,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(user).Error)

	svc := NewPaymentService(db, nil)

	// 创建测试订单
	tradeNo := generateTradeNo()
	order := &models.Order{
		UserID:  user.ID,
		Amount:  1000,
		Status:  "pending",
		TradeNo: tradeNo,
		PayType: "alipay",
	}
	require.NoError(t, db.Create(order).Error)

	// 测试更新为 success
	err := svc.UpdateOrderStatus(tradeNo, "success")
	require.NoError(t, err)

	updatedOrder, _ := svc.GetOrderByTradeNo(tradeNo)
	require.Equal(t, "success", updatedOrder.Status)

	// 测试更新为 failed
	err = svc.UpdateOrderStatus(tradeNo, "failed")
	require.NoError(t, err)

	updatedOrder, _ = svc.GetOrderByTradeNo(tradeNo)
	require.Equal(t, "failed", updatedOrder.Status)
}

func TestGetOrderStats(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建测试用户
	user := &models.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Balance:      0,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(user).Error)

	svc := NewPaymentService(db, nil)

	// 创建不同金额和状态的订单
	orders := []struct {
		amount int64
		status string
	}{
		{1000, "success"},
		{2000, "success"},
		{1500, "pending"},
		{3000, "failed"},
		{2500, "success"},
	}

	for _, o := range orders {
		order := &models.Order{
			UserID:  user.ID,
			Amount:  o.amount,
			Status:  o.status,
			TradeNo: generateTradeNo(),
			PayType: "alipay",
		}
		require.NoError(t, db.Create(order).Error)
	}

	// 测试订单统计
	orderCount, totalRevenue := svc.GetOrderStats()

	// 成功订单数：3 (1000 + 2000 + 2500)
	require.Equal(t, int64(3), orderCount)
	require.Equal(t, int64(5500), totalRevenue)
}

func TestCreatePackage(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc := NewPaymentService(db, nil)

	// 测试成功创建套餐
	pkg := &models.Package{
		Name:    "基础套餐",
		Traffic: 1024 * 1024 * 1024, // 1GB
		Price:   1000,               // 10元
	}
	err := svc.CreatePackage(pkg)
	require.NoError(t, err)
	require.NotZero(t, pkg.ID)

	// 验证创建成功
	foundPkg, err := svc.GetPackageByID(pkg.ID)
	require.NoError(t, err)
	require.Equal(t, "基础套餐", foundPkg.Name)
	require.Equal(t, int64(1024*1024*1024), foundPkg.Traffic)
	require.Equal(t, int64(1000), foundPkg.Price)
}

func TestListPackages(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc := NewPaymentService(db, nil)

	// 创建用户组
	userGroup := &models.UserGroup{
		Name: "VIP用户组",
	}
	require.NoError(t, db.Create(userGroup).Error)

	// 创建不同用户组的套餐
	packages := []models.Package{
		{Name: "通用套餐", Traffic: 1024, Price: 1000, UserGroupID: 0},
		{Name: "VIP套餐", Traffic: 1024*10, Price: 5000, UserGroupID: userGroup.ID},
		{Name: "VIP专属套餐", Traffic: 1024*20, Price: 8000, UserGroupID: userGroup.ID},
	}
	for _, p := range packages {
		require.NoError(t, svc.CreatePackage(&p))
	}

	// 测试获取所有套餐
	pkgs, err := svc.ListPackages(0)
	require.NoError(t, err)
	require.Len(t, pkgs, 3)

	// 测试按用户组筛选 (VIP用户组只能看到通用套餐 + VIP套餐)
	pkgs, err = svc.ListPackages(userGroup.ID)
	require.NoError(t, err)
	require.Len(t, pkgs, 2)
}

func TestUpdatePackage(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc := NewPaymentService(db, nil)

	// 创建套餐
	pkg := &models.Package{
		Name:    "原始套餐",
		Traffic: 1024,
		Price:   1000,
	}
	require.NoError(t, svc.CreatePackage(pkg))

	// 测试更新套餐
	updates := map[string]interface{}{
		"name":    "更新后的套餐",
		"traffic": 2048,
		"price":   1500,
	}
	err := svc.UpdatePackage(pkg.ID, updates)
	require.NoError(t, err)

	// 验证更新成功
	updatedPkg, err := svc.GetPackageByID(pkg.ID)
	require.NoError(t, err)
	require.Equal(t, "更新后的套餐", updatedPkg.Name)
	require.Equal(t, int64(2048), updatedPkg.Traffic)
	require.Equal(t, int64(1500), updatedPkg.Price)
}

func TestDeletePackage(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	svc := NewPaymentService(db, nil)

	// 创建套餐
	pkg := &models.Package{
		Name:    "待删除套餐",
		Traffic: 1024,
		Price:   1000,
	}
	require.NoError(t, svc.CreatePackage(pkg))

	// 测试删除套餐
	err := svc.DeletePackage(pkg.ID)
	require.NoError(t, err)

	// 验证删除成功
	_, err = svc.GetPackageByID(pkg.ID)
	require.Error(t, err)
	require.Equal(t, ErrPackageNotFound, err)
}

func TestCompleteOrder(t *testing.T) {
	db := setupTestDB(t)
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// 创建测试用户
	user := &models.User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Balance:      0,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, db.Create(user).Error)

	svc := NewPaymentService(db, nil)

	// 创建测试订单
	tradeNo := generateTradeNo()
	order := &models.Order{
		UserID:  user.ID,
		Amount:  1000,
		Status:  "pending",
		TradeNo: tradeNo,
		PayType: "alipay",
	}
	require.NoError(t, db.Create(order).Error)

	// 测试完成订单（带流量）
	err := svc.CompleteOrder(tradeNo, user.ID, 1024*1024*1024)
	require.NoError(t, err)

	// 验证订单状态
	completedOrder, _ := svc.GetOrderByTradeNo(tradeNo)
	require.Equal(t, "success", completedOrder.Status)

	// 验证用户余额更新
	var updatedUser models.User
	db.First(&updatedUser, user.ID)
	require.Equal(t, int64(1024*1024*1024), updatedUser.Balance)

	// 测试完成订单（不带流量）
	tradeNo2 := generateTradeNo()
	order2 := &models.Order{
		UserID:  user.ID,
		Amount:  2000,
		Status:  "pending",
		TradeNo: tradeNo2,
		PayType: "wechat",
	}
	require.NoError(t, db.Create(order2).Error)

	err = svc.CompleteOrder(tradeNo2, user.ID, 0)
	require.NoError(t, err)

	// 验证用户余额不变
	var updatedUser2 models.User
	db.First(&updatedUser2, user.ID)
	require.Equal(t, int64(1024*1024*1024), updatedUser2.Balance)
}

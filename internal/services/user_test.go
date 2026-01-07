package services

import (
	"testing"

	"bakaray/internal/models"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// createTestUserWithPassword creates a test user with password (used in handler tests)
func createTestUserWithPassword(t *testing.T, db *gorm.DB, username, password string, balance int64) *models.User {
	user := &models.User{
		Username: username,
		Balance:  balance,
		UserGroupID: 1,
		Role: "user",
	}

	hash := hashPassword(password)
	user.PasswordHash = hash

	err := db.Create(user).Error
	require.NoError(t, err)

	return user
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("成功创建用户", func(t *testing.T) {
		user, err := service.CreateUser("testuser", "password123", 1)

		require.NoError(t, err)
		require.NotNil(t, user)
		require.Equal(t, "testuser", user.Username)
		require.NotEmpty(t, user.PasswordHash)
		require.Equal(t, "user", user.Role)
		require.Equal(t, int64(0), user.Balance)
	})

	t.Run("用户名已存在时返回错误", func(t *testing.T) {
		// 先创建用户
		_, err := service.CreateUser("existinguser", "password123", 1)
		require.NoError(t, err)

		// 尝试创建同名用户
		_, err = service.CreateUser("existinguser", "anotherpassword", 1)
		require.Error(t, err)
		require.Equal(t, ErrUserExists, err)
	})

	t.Run("密码正确哈希存储", func(t *testing.T) {
		password := "mySecurePassword123"
		user, err := service.CreateUser("newhashuser", password, 1)

		require.NoError(t, err)
		require.NotNil(t, user)

		// 密码哈希不为空
		require.NotEmpty(t, user.PasswordHash)
		// 哈希值与原始密码不同
		require.NotEqual(t, password, user.PasswordHash)
		// 密码验证成功
		require.True(t, service.VerifyPassword(user, password))
		// 错误密码验证失败
		require.False(t, service.VerifyPassword(user, "wrongpassword"))
	})
}

func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("成功获取用户", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "testuser", "password", 1000)

		user, err := service.GetUserByID(createdUser.ID)

		require.NoError(t, err)
		require.NotNil(t, user)
		require.Equal(t, createdUser.ID, user.ID)
		require.Equal(t, "testuser", user.Username)
		require.Equal(t, int64(1000), user.Balance)
	})

	t.Run("用户不存在时返回错误", func(t *testing.T) {
		user, err := service.GetUserByID(99999)

		require.Error(t, err)
		require.Equal(t, ErrUserNotFound, err)
		require.Nil(t, user)
	})
}

func TestGetUserByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("成功获取用户", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "findme", "password", 500)

		user, err := service.GetUserByUsername("findme")

		require.NoError(t, err)
		require.NotNil(t, user)
		require.Equal(t, createdUser.ID, user.ID)
		require.Equal(t, "findme", user.Username)
	})

	t.Run("用户不存在时返回错误", func(t *testing.T) {
		user, err := service.GetUserByUsername("nonexistent")

		require.Error(t, err)
		require.Equal(t, ErrUserNotFound, err)
		require.Nil(t, user)
	})
}

func TestVerifyPassword(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("正确密码验证成功", func(t *testing.T) {
		password := "correctPassword123"
		createdUser := createTestUserWithPassword(t, db, "testuser", password, 0)

		result := service.VerifyPassword(createdUser, password)

		require.True(t, result)
	})

	t.Run("错误密码验证失败", func(t *testing.T) {
		password := "correctPassword123"
		wrongPassword := "wrongPassword456"
		createdUser := createTestUserWithPassword(t, db, "testuser", password, 0)

		result := service.VerifyPassword(createdUser, wrongPassword)

		require.False(t, result)
	})

	t.Run("空密码验证失败", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "testuser", "somepassword", 0)

		result := service.VerifyPassword(createdUser, "")

		require.False(t, result)
	})

	t.Run("空哈希验证失败", func(t *testing.T) {
		user := &models.User{
			ID:           1,
			Username:     "testuser",
			PasswordHash: "",
		}

		result := service.VerifyPassword(user, "anypassword")

		require.False(t, result)
	})
}

func TestUpdateBalance(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("增加余额", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "testuser", "password", 1000)
		initialBalance := createdUser.Balance

		err := service.UpdateBalance(createdUser.ID, 500)

		require.NoError(t, err)

		// 验证余额已增加
		updatedUser, err := service.GetUserByID(createdUser.ID)
		require.NoError(t, err)
		require.Equal(t, initialBalance+500, updatedUser.Balance)
	})

	t.Run("减少余额", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "testuser", "password", 1000)
		initialBalance := createdUser.Balance

		err := service.UpdateBalance(createdUser.ID, -300)

		require.NoError(t, err)

		// 验证余额已减少
		updatedUser, err := service.GetUserByID(createdUser.ID)
		require.NoError(t, err)
		require.Equal(t, initialBalance-300, updatedUser.Balance)
	})

	t.Run("余额可以为负数", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "testuser", "password", 100)

		err := service.UpdateBalance(createdUser.ID, -200)

		require.NoError(t, err)

		updatedUser, err := service.GetUserByID(createdUser.ID)
		require.NoError(t, err)
		require.Equal(t, int64(-100), updatedUser.Balance)
	})
}

func TestUpdateUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("成功更新用户信息", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "testuser", "password", 1000)

		updates := map[string]interface{}{
			"balance": 2000,
		}
		err := service.UpdateUser(createdUser.ID, updates)

		require.NoError(t, err)

		updatedUser, err := service.GetUserByID(createdUser.ID)
		require.NoError(t, err)
		require.Equal(t, int64(2000), updatedUser.Balance)
	})

	t.Run("使用 is_admin 更新角色为 admin", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "testuser", "password", 1000)

		updates := map[string]interface{}{
			"is_admin": true,
		}
		err := service.UpdateUser(createdUser.ID, updates)

		require.NoError(t, err)

		updatedUser, err := service.GetUserByID(createdUser.ID)
		require.NoError(t, err)
		require.Equal(t, "admin", updatedUser.Role)
		require.True(t, updatedUser.IsAdmin)
	})

	t.Run("使用 is_admin 更新角色为 user", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "testuser", "password", 1000)
		// 先设置为 admin
		db.Model(&models.User{}).Where("id = ?", createdUser.ID).Update("role", "admin")

		updates := map[string]interface{}{
			"is_admin": false,
		}
		err := service.UpdateUser(createdUser.ID, updates)

		require.NoError(t, err)

		updatedUser, err := service.GetUserByID(createdUser.ID)
		require.NoError(t, err)
		require.Equal(t, "user", updatedUser.Role)
	})
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("成功删除用户", func(t *testing.T) {
		createdUser := createTestUserWithPassword(t, db, "todelete", "password", 0)

		err := service.DeleteUser(createdUser.ID)

		require.NoError(t, err)

		// 验证用户已删除
		_, err = service.GetUserByID(createdUser.ID)
		require.Error(t, err)
		require.Equal(t, ErrUserNotFound, err)
	})

	t.Run("删除不存在的用户", func(t *testing.T) {
		err := service.DeleteUser(99999)

		require.Error(t, err)
	})
}

func TestListUsers(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("成功获取用户列表", func(t *testing.T) {
		// 创建多个用户
		for i := 1; i <= 5; i++ {
			createTestUserWithPassword(t, db, "testuser", "password", 0)
		}

		users, total := service.ListUsers(1, 10)

		require.Len(t, users, 5)
		require.Equal(t, int64(5), total)
	})

	t.Run("空列表", func(t *testing.T) {
		users, total := service.ListUsers(1, 10)

		require.Empty(t, users)
		require.Equal(t, int64(0), total)
	})

	t.Run("分页查询", func(t *testing.T) {
		// 创建10个用户
		for i := 1; i <= 10; i++ {
			createTestUserWithPassword(t, db, "pageuser", "password", 0)
		}

		// 获取第一页，每页3个
		users, total := service.ListUsers(1, 3)

		require.Len(t, users, 3)
		require.Equal(t, int64(10), total)

		// 获取第二页
		users2, _ := service.ListUsers(2, 3)
		require.Len(t, users2, 3)

		// 获取第四页（只有1个用户）
		users3, _ := service.ListUsers(4, 3)
		require.Len(t, users3, 1)
	})
}

func TestCountUsers(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("统计用户总数", func(t *testing.T) {
		// 创建3个用户
		for i := 1; i <= 3; i++ {
			createTestUserWithPassword(t, db, "countuser", "password", 0)
		}

		count := service.CountUsers()

		require.Equal(t, int64(3), count)
	})

	t.Run("统计空表", func(t *testing.T) {
		count := service.CountUsers()

		require.Equal(t, int64(0), count)
	})
}

func TestChangePassword(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("成功修改密码", func(t *testing.T) {
		oldPassword := "oldPassword123"
		newPassword := "newPassword456"
		createdUser := createTestUserWithPassword(t, db, "testuser", oldPassword, 0)

		err := service.ChangePassword(createdUser.ID, oldPassword, newPassword)

		require.NoError(t, err)

		// 验证新密码可以使用
		updatedUser, err := service.GetUserByID(createdUser.ID)
		require.NoError(t, err)
		require.True(t, service.VerifyPassword(updatedUser, newPassword))

		// 验证旧密码不能使用
		require.False(t, service.VerifyPassword(updatedUser, oldPassword))
	})

	t.Run("旧密码错误", func(t *testing.T) {
		oldPassword := "correctPassword"
		wrongPassword := "wrongPassword"
		newPassword := "newPassword"
		createdUser := createTestUserWithPassword(t, db, "testuser", oldPassword, 0)

		err := service.ChangePassword(createdUser.ID, wrongPassword, newPassword)

		require.Error(t, err)
		require.Equal(t, ErrInvalidPassword, err)

		// 原始密码仍然有效
		updatedUser, err := service.GetUserByID(createdUser.ID)
		require.NoError(t, err)
		require.True(t, service.VerifyPassword(updatedUser, oldPassword))
	})
}

func TestJWTAuthentication(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	redisClient := setupTestRedis(t)
	defer cleanupTestRedis(redisClient)

	service := NewUserService(db, redisClient)

	t.Run("JWT密钥配置正确", func(t *testing.T) {
		secret := service.GetJWTSecret()
		require.NotEmpty(t, secret)
	})

	t.Run("JWT过期时间配置正确", func(t *testing.T) {
		exp := service.GetJWTExpiration()
		require.Greater(t, exp, 0)
		require.Equal(t, 86400, exp) // 默认24小时
	})
}

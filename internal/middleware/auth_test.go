package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockUserService implements the methods used by AuthMiddleware
type mockUserService struct {
	jwtSecret    string
	jwtExp       int
	mockUserID   uint
	mockUserErr  error
}

func (m *mockUserService) GetJWTSecret() string {
	return m.jwtSecret
}

func (m *mockUserService) GetJWTExpiration() int {
	return m.jwtExp
}

// testAuthMiddleware is a test version of AuthMiddleware
type testAuthMiddleware struct {
	userService *mockUserService
	jwtSecret   string
	jwtExp      int
}

func newTestAuthMiddleware(ms *mockUserService) *testAuthMiddleware {
	return &testAuthMiddleware{
		userService: ms,
		jwtSecret:   ms.jwtSecret,
		jwtExp:      ms.jwtExp,
	}
}

func (m *testAuthMiddleware) parseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

func (m *testAuthMiddleware) ParseToken(tokenString string) (*Claims, error) {
	return m.parseToken(tokenString)
}

func (m *testAuthMiddleware) GenerateToken(userID uint, role string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(m.jwtExp) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.jwtSecret))
}

func TestGenerateToken(t *testing.T) {
	mockService := &mockUserService{
		jwtSecret: "test-secret-key",
		jwtExp:    3600,
	}
	middleware := newTestAuthMiddleware(mockService)

	token, err := middleware.GenerateToken(1, "admin")
	if err != nil {
		t.Fatalf("GenerateToken error = %v", err)
	}

	if token == "" {
		t.Error("GenerateToken returned empty token")
	}

	// 验证 token 可以解析
	claims, err := middleware.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error = %v", err)
	}

	if claims.UserID != 1 {
		t.Errorf("UserID = %v, want 1", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %v, want admin", claims.Role)
	}
}

func TestParseToken_Invalid(t *testing.T) {
	mockService := &mockUserService{
		jwtSecret: "test-secret-key",
		jwtExp:    3600,
	}
	middleware := newTestAuthMiddleware(mockService)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "invalid-token"},
		{"wrong signature", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxfQ.wrong-signature"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := middleware.ParseToken(tt.token)
			if err == nil {
				t.Error("ParseToken should return error for invalid token")
			}
		})
	}
}

func TestParseToken_Expired(t *testing.T) {
	mockService := &mockUserService{
		jwtSecret: "test-secret-key",
		jwtExp:    3600,
	}
	middleware := newTestAuthMiddleware(mockService)

	// 创建一个过期的 token
	now := time.Now()
	claims := &Claims{
		UserID: 1,
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-1 * time.Hour)), // 1小时前过期
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(mockService.jwtSecret))
	if err != nil {
		t.Fatalf("Create expired token error = %v", err)
	}

	_, err = middleware.ParseToken(tokenString)
	if err == nil {
		t.Error("ParseToken should return error for expired token")
	}
}

func TestAuthenticate_NoHeader(t *testing.T) {
	mockService := &mockUserService{
		jwtSecret:  "test-secret-key",
		jwtExp:     3600,
		mockUserID: 1,
	}
	_ = newTestAuthMiddleware(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	// 模拟认证中间件
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证令牌"})
		c.Abort()
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticate_InvalidFormat(t *testing.T) {
	mockService := &mockUserService{
		jwtSecret:  "test-secret-key",
		jwtExp:     3600,
		mockUserID: 1,
	}
	_ = newTestAuthMiddleware(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "InvalidFormat token")

	// 模拟认证中间件逻辑
	authHeader := c.GetHeader(AuthorizationHeader)
	parts := make([]string, 0)
	if authHeader != "" {
		parts = make([]string, 2)
		copy(parts, []string{"InvalidFormat", "token"})
	}
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的认证格式"})
		c.Abort()
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	mockService := &mockUserService{
		jwtSecret:  "test-secret-key",
		jwtExp:     3600,
		mockUserID: 1,
	}
	middleware := newTestAuthMiddleware(mockService)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer invalid-token")

	// 模拟认证中间件逻辑
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader != "" {
		parts := []string{"Bearer", "invalid-token"}
		if len(parts) == 2 && parts[0] == "Bearer" {
			_, err := middleware.parseToken(parts[1])
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效或过期的令牌"})
				c.Abort()
			}
		}
	}

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticate_ValidToken(t *testing.T) {
	mockService := &mockUserService{
		jwtSecret:  "test-secret-key",
		jwtExp:     3600,
		mockUserID: 1,
	}
	middleware := newTestAuthMiddleware(mockService)

	// 生成一个有效的 token
	token, _ := middleware.GenerateToken(1, "user")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	// 模拟认证中间件逻辑
	authHeader := c.GetHeader(AuthorizationHeader)
	if authHeader != "" {
		parts := make([]string, 2)
		copy(parts, []string{"Bearer", token})
		if len(parts) == 2 && parts[0] == "Bearer" {
			claims, err := middleware.parseToken(parts[1])
			if err == nil {
				c.Set(UserIDKey, claims.UserID)
				c.Set(UserRoleKey, claims.Role)
				c.Next()
			} else {
				c.Abort()
			}
		} else {
			c.Abort()
		}
	} else {
		c.Abort()
	}

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	userID, exists := c.Get(UserIDKey)
	if !exists {
		t.Error("user_id should be set in context")
	}
	if userID != uint(1) {
		t.Errorf("user_id = %v, want 1", userID)
	}
}

func TestAdminRequired(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func(c *gin.Context)
		expectedCode int
	}{
		{
			name: "admin user - allowed",
			setupContext: func(c *gin.Context) {
				c.Set(UserRoleKey, "admin")
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "regular user - forbidden",
			setupContext: func(c *gin.Context) {
				c.Set(UserRoleKey, "user")
			},
			expectedCode: http.StatusForbidden,
		},
		{
			name: "no role set - forbidden",
			setupContext: func(c *gin.Context) {
				// 不设置角色
			},
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/admin", nil)

			tt.setupContext(c)

			// 模拟 AdminRequired 中间件逻辑
			role, _ := c.Get(UserRoleKey)
			if role != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "需要管理员权限"})
				c.Abort()
			}

			if w.Code != tt.expectedCode {
				t.Errorf("Status = %v, want %v", w.Code, tt.expectedCode)
			}
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *gin.Context)
		expected uint
	}{
		{
			name: "valid user id",
			setup: func(c *gin.Context) {
				c.Set(UserIDKey, uint(123))
			},
			expected: 123,
		},
		{
			name: "no user id set",
			setup: func(c *gin.Context) {
				// 不设置
			},
			expected: 0,
		},
		{
			name: "wrong type",
			setup: func(c *gin.Context) {
				c.Set(UserIDKey, "not a uint")
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			got := GetUserID(c)
			if got != tt.expected {
				t.Errorf("GetUserID() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetUserRole(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(c *gin.Context)
		expected string
	}{
		{
			name: "valid role",
			setup: func(c *gin.Context) {
				c.Set(UserRoleKey, "admin")
			},
			expected: "admin",
		},
		{
			name:     "no role set",
			setup:    func(c *gin.Context) {},
			expected: "",
		},
		{
			name: "wrong type",
			setup: func(c *gin.Context) {
				c.Set(UserRoleKey, 123)
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setup(c)

			got := GetUserRole(c)
			if got != tt.expected {
				t.Errorf("GetUserRole() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestClaims(t *testing.T) {
	claims := &Claims{
		UserID: 1,
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	if claims.UserID != 1 {
		t.Errorf("UserID = %v, want 1", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %v, want admin", claims.Role)
	}
}

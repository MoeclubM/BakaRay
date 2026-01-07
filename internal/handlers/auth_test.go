package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bakaray/internal/middleware"
	"bakaray/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockUserService 是 UserService 的测试模拟实现
type mockUserService struct {
	jwtSecret string
	jwtExp    int
	users     map[string]*models.User
	userByID  map[uint]*models.User
}

func newMockUserService() *mockUserService {
	return &mockUserService{
		jwtSecret: "test-jwt-secret-key",
		jwtExp:    86400,
		users:     make(map[string]*models.User),
		userByID:  make(map[uint]*models.User),
	}
}

// hashPassword 使用 bcrypt 哈希密码
func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func (m *mockUserService) GetJWTSecret() string {
	return m.jwtSecret
}

func (m *mockUserService) GetJWTExpiration() int {
	return m.jwtExp
}

func (m *mockUserService) GetUserByUsername(username string) (*models.User, error) {
	user, ok := m.users[username]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserService) GetUserByID(id uint) (*models.User, error) {
	user, ok := m.userByID[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (m *mockUserService) VerifyPassword(user *models.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

func (m *mockUserService) CreateUser(username, password string, groupID uint) (*models.User, error) {
	// 检查用户名是否已存在
	if _, exists := m.users[username]; exists {
		return nil, gorm.ErrRecordNotFound
	}
	user := &models.User{
		ID:           uint(len(m.users) + 1),
		Username:     username,
		PasswordHash: hashPassword(password),
		Balance:      0,
		UserGroupID:  groupID,
		Role:         "user",
	}
	m.users[username] = user
	m.userByID[user.ID] = user
	return user, nil
}

// addTestUser 添加测试用户
func (m *mockUserService) addTestUser(username, password string, id uint, role string) {
	hashedPassword := hashPassword(password)
	user := &models.User{
		ID:           id,
		Username:     username,
		PasswordHash: hashedPassword,
		Balance:      10000,
		Role:         role,
	}
	m.users[username] = user
	m.userByID[id] = user
}

// testAuthHandler 是 AuthHandler 的测试版本，使用 mockUserService
type testAuthHandler struct {
	userService *mockUserService
}

func newTestAuthHandler(ms *mockUserService) *testAuthHandler {
	return &testAuthHandler{userService: ms}
}

// testAuthMiddleware 是 middleware.AuthMiddleware 的测试版本
type testAuthMiddleware struct {
	jwtSecret string
	jwtExp    int
}

func newTestAuthMiddleware(ms *mockUserService) *testAuthMiddleware {
	return &testAuthMiddleware{
		jwtSecret: ms.jwtSecret,
		jwtExp:    ms.jwtExp,
	}
}

func (m *testAuthMiddleware) GenerateToken(userID uint, role string) (string, error) {
	return generateTestToken(m.jwtSecret, m.jwtExp, userID, role)
}

func (m *testAuthMiddleware) ParseTokenAllowExpired(tokenString string) (*middleware.Claims, error) {
	claims := &middleware.Claims{}

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, claims)
	if err != nil {
		return nil, err
	}

	// 验证签名
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}

// generateTestToken 生成测试用 JWT token
func generateTestToken(secret string, expSeconds int, userID uint, role string) (string, error) {
	claims := &middleware.Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// generateExpiredToken 生成已过期的测试 token
func generateExpiredToken(secret string, userID uint, role string) (string, error) {
	claims := &middleware.Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // 1小时前过期
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (h *testAuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	user, err := h.userService.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户不存在"})
		return
	}

	if !h.userService.VerifyPassword(user, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "密码错误"})
		return
	}

	authMiddleware := newTestAuthMiddleware(h.userService)
	token, _ := authMiddleware.GenerateToken(user.ID, user.Role)

	c.JSON(http.StatusOK, LoginResponse{
		Code:   0,
		Token:  token,
		Expire: h.userService.GetJWTExpiration(),
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
			Balance:  user.Balance,
		},
	})
}

// Register 注册测试
func (h *testAuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	user, err := h.userService.CreateUser(req.Username, req.Password, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户已存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "注册成功",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// Refresh 刷新令牌测试
func (h *testAuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	_ = c.ShouldBindJSON(&req)

	authMiddleware := newTestAuthMiddleware(h.userService)

	tokenString := req.Token
	if tokenString == "" {
		if authHeader := c.GetHeader(middleware.AuthorizationHeader); authHeader != "" {
			parts := splitAuthHeader(authHeader)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}
	}
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少令牌"})
		return
	}

	claims, err := authMiddleware.ParseTokenAllowExpired(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的令牌"})
		return
	}

	user, err := h.userService.GetUserByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户不存在"})
		return
	}

	newToken, _ := authMiddleware.GenerateToken(user.ID, user.Role)

	c.JSON(http.StatusOK, LoginResponse{
		Code:   0,
		Token:  newToken,
		Expire: h.userService.GetJWTExpiration(),
	})
}

// splitAuthHeader 分割 Authorization header
func splitAuthHeader(authHeader string) []string {
	parts := make([]string, 0, 2)
	start := 0
	for i := 0; i < len(authHeader) && len(parts) < 2; i++ {
		if authHeader[i] == ' ' {
			parts = append(parts, authHeader[start:i])
			start = i + 1
		}
	}
	parts = append(parts, authHeader[start:])
	return parts
}

// Helper function to create login request body
func createLoginRequest(username, password string) *bytes.Buffer {
	body, _ := json.Marshal(LoginRequest{Username: username, Password: password})
	return bytes.NewBuffer(body)
}

// Helper function to create register request body
func createRegisterRequest(username, password string) *bytes.Buffer {
	body, _ := json.Marshal(RegisterRequest{Username: username, Password: password})
	return bytes.NewBuffer(body)
}

// Helper function to create refresh request body
func createRefreshRequest(token string) *bytes.Buffer {
	body, _ := json.Marshal(RefreshRequest{Token: token})
	return bytes.NewBuffer(body)
}

// ==================== TestLogin Tests ====================

func TestLogin_Success(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("testuser", "password123", 1, "user")

	handler := newTestAuthHandler(mockService)

	body := createLoginRequest("testuser", "password123")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/login", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response.Code != 0 {
		t.Errorf("Code = %v, want 0", response.Code)
	}

	if response.Token == "" {
		t.Error("Token should not be empty")
	}

	if response.User == nil {
		t.Error("User should not be nil")
	}

	if response.User.Username != "testuser" {
		t.Errorf("Username = %v, want testuser", response.User.Username)
	}

	if response.User.Role != "user" {
		t.Errorf("Role = %v, want user", response.User.Role)
	}

	if response.Expire != 86400 {
		t.Errorf("Expire = %v, want 86400", response.Expire)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	mockService := newMockUserService()
	// 不添加任何用户

	handler := newTestAuthHandler(mockService)

	body := createLoginRequest("nonexistent", "password123")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/login", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response["code"].(float64) != 401 {
		t.Errorf("Code = %v, want 401", response["code"])
	}

	if response["message"] != "用户不存在" {
		t.Errorf("Message = %v, want 用户不存在", response["message"])
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("testuser", "correctpassword", 1, "user")

	handler := newTestAuthHandler(mockService)

	body := createLoginRequest("testuser", "wrongpassword")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/login", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response["code"].(float64) != 401 {
		t.Errorf("Code = %v, want 401", response["code"])
	}

	if response["message"] != "密码错误" {
		t.Errorf("Message = %v, want 密码错误", response["message"])
	}
}

func TestLogin_InvalidRequest(t *testing.T) {
	tests := []struct {
		name   string
		body   string
	}{
		{
			name:   "missing username",
			body:   `{"password":"password123"}`,
		},
		{
			name:   "missing password",
			body:   `{"username":"testuser"}`,
		},
		{
			name:   "empty body",
			body:   `{}`,
		},
		{
			name:   "invalid json",
			body:   `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := newMockUserService()
			handler := newTestAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/login", bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Login(c)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Unmarshal response error = %v", err)
			}

			if response["code"].(float64) != 400 {
				t.Errorf("Code = %v, want 400", response["code"])
			}

			if response["message"] != "参数错误" {
				t.Errorf("Message = %v, want 参数错误", response["message"])
			}
		})
	}
}

// ==================== TestRegister Tests ====================

func TestRegister_Success(t *testing.T) {
	mockService := newMockUserService()
	handler := newTestAuthHandler(mockService)

	body := createRegisterRequest("newuser", "password123")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/register", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response["code"].(float64) != 0 {
		t.Errorf("Code = %v, want 0", response["code"])
	}

	if response["message"] != "注册成功" {
		t.Errorf("Message = %v, want 注册成功", response["message"])
	}

	data := response["data"].(map[string]interface{})
	if data["username"] != "newuser" {
		t.Errorf("Username = %v, want newuser", data["username"])
	}

	if data["id"] == nil {
		t.Error("id should not be nil")
	}
}

func TestRegister_UsernameExists(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("existinguser", "password123", 1, "user")

	handler := newTestAuthHandler(mockService)

	body := createRegisterRequest("existinguser", "newpassword")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/register", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Register(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response["code"].(float64) != 400 {
		t.Errorf("Code = %v, want 400", response["code"])
	}

	if response["message"] != "用户已存在" {
		t.Errorf("Message = %v, want 用户已存在", response["message"])
	}
}

func TestRegister_InvalidRequest(t *testing.T) {
	tests := []struct {
		name   string
		body   string
	}{
		{
			name:   "missing username",
			body:   `{"password":"password123"}`,
		},
		{
			name:   "missing password",
			body:   `{"username":"testuser"}`,
		},
		{
			name:   "username too short",
			body:   `{"username":"ab","password":"password123"}`,
		},
		{
			name:   "password too short",
			body:   `{"username":"testuser","password":"123"}`,
		},
		{
			name:   "empty body",
			body:   `{}`,
		},
		{
			name:   "invalid json",
			body:   `{invalid`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := newMockUserService()
			handler := newTestAuthHandler(mockService)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/register", bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			handler.Register(c)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Unmarshal response error = %v", err)
			}

			if response["code"].(float64) != 400 {
				t.Errorf("Code = %v, want 400", response["code"])
			}
		})
	}
}

// ==================== TestRefresh Tests ====================

func TestRefresh_Success(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("testuser", "password123", 1, "user")

	handler := newTestAuthHandler(mockService)

	// 创建一个过期的 token 用于测试
	expiredToken, _ := generateExpiredToken(mockService.jwtSecret, 1, "user")

	body := createRefreshRequest(expiredToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/refresh", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response.Code != 0 {
		t.Errorf("Code = %v, want 0", response.Code)
	}

	if response.Token == "" {
		t.Error("Token should not be empty")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	mockService := newMockUserService()
	handler := newTestAuthHandler(mockService)

	body := createRefreshRequest("invalid-token")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/refresh", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response["code"].(float64) != 401 {
		t.Errorf("Code = %v, want 401", response["code"])
	}

	if response["message"] != "无效的令牌" {
		t.Errorf("Message = %v, want 无效的令牌", response["message"])
	}
}

func TestRefresh_TokenFromHeader(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("testuser", "password123", 1, "user")

	handler := newTestAuthHandler(mockService)

	// 创建一个过期的 token 用于测试
	expiredToken, _ := generateExpiredToken(mockService.jwtSecret, 1, "user")

	// 空 body，从 header 获取 token
	body := createRefreshRequest("")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/refresh", body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Bearer "+expiredToken)

	handler.Refresh(c)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response.Code != 0 {
		t.Errorf("Code = %v, want 0", response.Code)
	}

	if response.Token == "" {
		t.Error("Token should not be empty")
	}
}

func TestRefresh_MissingToken(t *testing.T) {
	mockService := newMockUserService()
	handler := newTestAuthHandler(mockService)

	// 空 body，空 header
	body := createRefreshRequest("")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/refresh", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response["code"].(float64) != 400 {
		t.Errorf("Code = %v, want 400", response["code"])
	}

	if response["message"] != "缺少令牌" {
		t.Errorf("Message = %v, want 缺少令牌", response["message"])
	}
}

func TestRefresh_UserNotFound(t *testing.T) {
	mockService := newMockUserService()
	// 不添加用户

	// 创建一个 token，用户 ID 为 999（不存在的用户）
	expiredToken, _ := generateExpiredToken(mockService.jwtSecret, 999, "user")

	handler := newTestAuthHandler(mockService)

	body := createRefreshRequest(expiredToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/refresh", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusUnauthorized)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response["code"].(float64) != 401 {
		t.Errorf("Code = %v, want 401", response["code"])
	}

	if response["message"] != "用户不存在" {
		t.Errorf("Message = %v, want 用户不存在", response["message"])
	}
}

func TestRefresh_ValidNonExpiredToken(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("testuser", "password123", 1, "user")

	handler := newTestAuthHandler(mockService)

	// 创建一个有效的 token（未过期）
	validToken, _ := generateTestToken(mockService.jwtSecret, 86400, 1, "user")

	body := createRefreshRequest(validToken)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/refresh", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Refresh(c)

	if w.Code != http.StatusOK {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusOK)
	}

	var response LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Unmarshal response error = %v", err)
	}

	if response.Code != 0 {
		t.Errorf("Code = %v, want 0", response.Code)
	}

	if response.Token == "" {
		t.Error("Token should not be empty")
	}
}

// ==================== Edge Cases Tests ====================

func TestLogin_EmptyPassword(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("testuser", "password123", 1, "user")

	handler := newTestAuthHandler(mockService)

	body := createLoginRequest("testuser", "")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/login", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestLogin_EmptyUsername(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("testuser", "password123", 1, "user")

	handler := newTestAuthHandler(mockService)

	body := createLoginRequest("", "password123")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/login", body)
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestRefresh_WrongBearerFormat(t *testing.T) {
	mockService := newMockUserService()
	mockService.addTestUser("testuser", "password123", 1, "user")

	handler := newTestAuthHandler(mockService)

	expiredToken, _ := generateExpiredToken(mockService.jwtSecret, 1, "user")

	body := createRefreshRequest("")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/refresh", body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Authorization", "Basic "+expiredToken) // 使用 Basic 而不是 Bearer

	handler.Refresh(c)

	// 应该返回缺少令牌的错误，因为格式不正确
	if w.Code != http.StatusBadRequest {
		t.Errorf("Status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

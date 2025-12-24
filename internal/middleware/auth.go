package middleware

import (
	"net/http"
	"strings"
	"time"

	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	AuthorizationHeader = "Authorization"
	UserIDKey           = "user_id"
	UserRoleKey         = "user_role"
)

// Claims JWT claims
type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	userService *services.UserService
	jwtSecret   string
	jwtExp      int
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(userService *services.UserService) *AuthMiddleware {
	return &AuthMiddleware{
		userService: userService,
		jwtSecret:   userService.GetJWTSecret(),
		jwtExp:      userService.GetJWTExpiration(),
	}
}

// Authenticate 用户认证
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证令牌"})
			c.Abort()
			return
		}

		// Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的认证格式"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := m.parseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效或过期的令牌"})
			c.Abort()
			return
		}

		// 验证用户是否存在
		user, err := m.userService.GetUserByID(claims.UserID)
		if err != nil || user.ID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户不存在"})
			c.Abort()
			return
		}

		// 设置用户信息到上下文
		c.Set(UserIDKey, claims.UserID)
		c.Set(UserRoleKey, claims.Role)

		c.Next()
	}
}

// AdminRequired 管理员权限验证
func (m *AuthMiddleware) AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(UserRoleKey)
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ParseToken 解析 JWT token (公开方法)
func (m *AuthMiddleware) ParseToken(tokenString string) (*Claims, error) {
	return m.parseToken(tokenString)
}

// ParseTokenAllowExpired parses and validates token signature, but allows an expired token
// within a short grace window for refresh purposes.
func (m *AuthMiddleware) ParseTokenAllowExpired(tokenString string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, err := parser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.jwtSecret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	// Allow refresh for tokens expired within 7 days.
	if claims.ExpiresAt != nil {
		expiredFor := time.Since(claims.ExpiresAt.Time)
		if expiredFor > 7*24*time.Hour {
			return nil, jwt.ErrTokenExpired
		}
	}

	return claims, nil
}

// parseToken 解析 JWT token (私有方法)
func (m *AuthMiddleware) parseToken(tokenString string) (*Claims, error) {
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

// GenerateToken 生成 JWT token
func (m *AuthMiddleware) GenerateToken(userID uint, role string) (string, error) {
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

// GetUserID 从上下文获取用户ID
func GetUserID(c *gin.Context) uint {
	if id, exists := c.Get(UserIDKey); exists {
		if uid, ok := id.(uint); ok {
			return uid
		}
	}
	return 0
}

// GetUserRole 从上下文获取用户角色
func GetUserRole(c *gin.Context) string {
	if role, exists := c.Get(UserRoleKey); exists {
		if r, ok := role.(string); ok {
			return r
		}
	}
	return ""
}

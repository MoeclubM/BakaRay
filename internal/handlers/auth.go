package handlers

import (
	"net/http"
	"strings"

	"bakaray/internal/middleware"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	userService *services.UserService
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(userService *services.UserService) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Code   int    `json:"code"`
	Token  string `json:"token"`
	Expire int    `json:"expire"`
}

// Login 登录
func (h *AuthHandler) Login(c *gin.Context) {
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

	authMiddleware := middleware.NewAuthMiddleware(h.userService)
	token, err := authMiddleware.GenerateToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Code:   0,
		Token:  token,
		Expire: h.userService.GetJWTExpiration(),
	})
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=6"`
}

// Register 注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	user, err := h.userService.CreateUser(req.Username, req.Password, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
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

// RefreshRequest 刷新令牌请求
type RefreshRequest struct {
	Token string `json:"token"`
}

// Refresh 刷新令牌
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	_ = c.ShouldBindJSON(&req)

	authMiddleware := middleware.NewAuthMiddleware(h.userService)

	tokenString := strings.TrimSpace(req.Token)
	if tokenString == "" {
		// Fallback to Authorization header.
		if authHeader := c.GetHeader(middleware.AuthorizationHeader); authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}
	}
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少令牌"})
		return
	}

	// 解析旧 token
	claims, err := authMiddleware.ParseTokenAllowExpired(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的令牌"})
		return
	}

	// 验证用户是否存在
	user, err := h.userService.GetUserByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户不存在"})
		return
	}

	// 生成新 token
	newToken, err := authMiddleware.GenerateToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Code:   0,
		Token:  newToken,
		Expire: h.userService.GetJWTExpiration(),
	})
}

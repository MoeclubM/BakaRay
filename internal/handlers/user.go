package handlers

import (
	"net/http"
	"strconv"
	"time"

	"bakaray/internal/logger"
	"bakaray/internal/middleware"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService *services.UserService
	ruleService *services.RuleService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *services.UserService, ruleService *services.RuleService) *UserHandler {
	return &UserHandler{
		userService: userService,
		ruleService: ruleService,
	}
}

// GetProfile 获取当前用户信息
func (h *UserHandler) GetProfile(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "user")

	log.Debug("GetProfile request")

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		logger.Error("GetProfile: user not found", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	log.Info("GetProfile success", "username", user.Username)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":          user.ID,
			"username":    user.Username,
			"balance":     user.Balance,
			"user_group":  user.UserGroupID,
			"role":        user.Role,
			"created_at":  user.CreatedAt,
		},
	})
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Username string `json:"username"`
}

// UpdateProfile 更新用户信息
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "user")

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("UpdateProfile: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	updates := make(map[string]interface{})
	if req.Username != "" {
		updates["username"] = req.Username
	}

	if len(updates) == 0 {
		logger.Warn("UpdateProfile: no updates", "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有可更新的内容"})
		return
	}

	log.Debug("UpdateProfile request", "updates", updates)

	if err := h.userService.UpdateUser(userID, updates); err != nil {
		logger.Error("UpdateProfile: update failed", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	log.Info("UpdateProfile success", "updates", updates)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// GetTrafficStats 获取流量统计
func (h *UserHandler) GetTrafficStats(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "user")

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	log.Debug("GetTrafficStats request", "days", days)

	usedTotal, err := h.ruleService.GetUserTrafficUsed(userID)
	if err != nil {
		logger.Error("GetTrafficStats: get used traffic failed", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取统计失败"})
		return
	}

	bytesIn, bytesOut, err := h.ruleService.GetUserTrafficStats(userID, since)
	if err != nil {
		logger.Error("GetTrafficStats: get traffic stats failed", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取统计失败"})
		return
	}

	user, _ := h.userService.GetUserByID(userID)
	remaining := int64(0)
	if user != nil {
		remaining = user.Balance - usedTotal
		if remaining < 0 {
			remaining = 0
		}
	}

	log.Info("GetTrafficStats success", "days", days, "bytes_in", bytesIn, "bytes_out", bytesOut, "total_used", usedTotal, "remaining", remaining)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"days":            days,
			"bytes_in":        bytesIn,
			"bytes_out":       bytesOut,
			"total_used":      usedTotal,
			"remaining_bytes": remaining,
		},
	})
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword 修改密码
func (h *UserHandler) ChangePassword(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "user")

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("ChangePassword: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("ChangePassword request")

	if err := h.userService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		if err == services.ErrInvalidPassword {
			logger.Warn("ChangePassword: invalid old password", "user_id", userID, "request_id", requestID)
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "原密码错误"})
			return
		}
		logger.Error("ChangePassword: change password failed", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "修改失败"})
		return
	}

	log.Info("ChangePassword success")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码修改成功",
	})
}

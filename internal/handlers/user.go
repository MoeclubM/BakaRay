package handlers

import (
	"net/http"
	"strconv"
	"time"

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
	userID := middleware.GetUserID(c)
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"balance":    user.Balance,
			"user_group": user.UserGroupID,
			"role":       user.Role,
			"created_at": user.CreatedAt,
		},
	})
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Username string `json:"username"`
}

// UpdateProfile 更新用户信息
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	updates := make(map[string]interface{})
	if req.Username != "" {
		updates["username"] = req.Username
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有可更新的内容"})
		return
	}

	// 更新用户信息（不更新密码和角色）
	if err := h.userService.UpdateUser(userID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// GetTrafficStats 获取流量统计
func (h *UserHandler) GetTrafficStats(c *gin.Context) {
	userID := middleware.GetUserID(c)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	usedTotal, err := h.ruleService.GetUserTrafficUsed(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取统计失败"})
		return
	}

	bytesIn, bytesOut, err := h.ruleService.GetUserTrafficStats(userID, since)
	if err != nil {
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

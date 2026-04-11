package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakaray/internal/middleware"
	"bakaray/internal/models"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createUserHandlerForTest(t *testing.T) (*UserHandler, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.UserGroup{}, &models.User{}))

	return NewUserHandler(services.NewUserService(db, nil), nil, services.NewUserGroupService(db)), db
}

func TestGetProfile_ReturnsUserGroupName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler, db := createUserHandlerForTest(t)

	group := models.UserGroup{
		Name:        "正式用户组",
		Description: "用于测试仪表盘展示",
	}
	require.NoError(t, db.Create(&group).Error)

	user := models.User{
		Username:       "tester",
		PasswordHash:   "hash",
		Balance:        1200,
		TrafficBalance: 4096,
		UserGroupID:    group.ID,
		Role:           "user",
	}
	require.NoError(t, db.Create(&user).Error)

	router := gin.New()
	router.GET("/profile", func(c *gin.Context) {
		c.Set(middleware.UserIDKey, user.ID)
		handler.GetProfile(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Username      string `json:"username"`
			UserGroupID   uint   `json:"user_group_id"`
			UserGroupName string `json:"user_group_name"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, user.Username, resp.Data.Username)
	require.Equal(t, group.ID, resp.Data.UserGroupID)
	require.Equal(t, group.Name, resp.Data.UserGroupName)
}

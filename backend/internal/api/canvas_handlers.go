package api

import (
	"net/http"
	"time"

	"google-ai-proxy/internal/db"

	"github.com/gin-gonic/gin"
)

// ListCanvasProjects GET /api/canvas
func ListCanvasProjects(c *gin.Context) {
	userID := c.GetUint64("userID")
	limit, offset := parseListPagination(c)
	var projects []db.CanvasProject
	db.DB.Select("id, user_id, name, status, created_at, updated_at").
		Where("user_id = ? AND status = 'active'", userID).
		Order("updated_at DESC").Limit(limit).Offset(offset).Find(&projects)
	var total int64
	db.DB.Model(&db.CanvasProject{}).Where("user_id = ? AND status = 'active'", userID).Count(&total)
	c.JSON(http.StatusOK, gin.H{"items": projects, "total": total})
}

// CreateCanvasProject POST /api/canvas
func CreateCanvasProject(c *gin.Context) {
	userID := c.GetUint64("userID")
	var req struct {
		Name   string `json:"name"`
		Layout string `json:"layout"`
	}
	c.ShouldBindJSON(&req)
	if req.Name == "" {
		req.Name = "未命名画布"
	}
	p := &db.CanvasProject{
		UserID: userID,
		Name:   req.Name,
		Layout: req.Layout,
		Status: "active",
	}
	if err := db.DB.Create(p).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// GetCanvasProject GET /api/canvas/:id
func GetCanvasProject(c *gin.Context) {
	userID := c.GetUint64("userID")
	id := parseUintParam(c.Param("id"))
	var p db.CanvasProject
	if err := db.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "画布不存在"})
		return
	}
	if p.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// UpdateCanvasProject PUT /api/canvas/:id
func UpdateCanvasProject(c *gin.Context) {
	userID := c.GetUint64("userID")
	id := parseUintParam(c.Param("id"))
	var p db.CanvasProject
	if err := db.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "画布不存在"})
		return
	}
	if p.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var req struct {
		Name   *string `json:"name"`
		Layout *string `json:"layout"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Layout != nil {
		updates["layout"] = *req.Layout
	}
	db.DB.Model(&p).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "已保存"})
}

// DeleteCanvasProject DELETE /api/canvas/:id
func DeleteCanvasProject(c *gin.Context) {
	userID := c.GetUint64("userID")
	id := parseUintParam(c.Param("id"))
	var p db.CanvasProject
	if err := db.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "画布不存在"})
		return
	}
	if p.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	db.DB.Delete(&p)
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

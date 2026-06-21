package api

import (
	"io"
	"net/http"
	"strings"

	"google-ai-proxy/internal/db"
	"google-ai-proxy/internal/novel"

	"github.com/gin-gonic/gin"
)

const novelMaxUploadBytes = 10 * 1024 * 1024 // 10MB

// UploadNovel POST /api/novel/upload  multipart file=.txt
func UploadNovel(c *gin.Context) {
	userID := c.GetUint64("userID")
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传 .txt 文件"})
		return
	}
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".txt") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 .txt 文件"})
		return
	}
	if file.Size > novelMaxUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大（上限 10MB）"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}
	content := string(raw)

	chapters := novel.ParseChapters(content)
	title := strings.TrimSuffix(file.Filename, ".txt")
	if title == "" {
		title = "未命名小说"
	}

	n := &db.Novel{
		UserID:         userID,
		Title:          title,
		SourceFilename: file.Filename,
		RawContent:     content,
		ChapterCount:   len(chapters),
		Status:         "active",
	}
	if err := db.DB.Create(n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	rows := make([]db.NovelChapter, 0, len(chapters))
	for _, ch := range chapters {
		rows = append(rows, db.NovelChapter{
			NovelID:          n.ID,
			ChapterIndex:     ch.Index,
			Title:            ch.Title,
			Content:          ch.Content,
			StoryboardStatus: "none",
		})
	}
	if len(rows) > 0 {
		if err := db.DB.Create(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存章节失败"})
			return
		}
	}

	resp := gin.H{"novel_id": n.ID, "title": n.Title, "chapter_count": n.ChapterCount}
	if len(chapters) == 1 && chapters[0].Title == "全文" {
		resp["warning"] = "未识别到章节标题，已按整文处理"
	}
	c.JSON(http.StatusOK, resp)
}

// ListNovels GET /api/novel
func ListNovels(c *gin.Context) {
	userID := c.GetUint64("userID")
	limit, offset := parseListPagination(c)
	var novels []db.Novel
	db.DB.Where("user_id = ? AND status = 'active'", userID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&novels)
	var total int64
	db.DB.Model(&db.Novel{}).Where("user_id = ? AND status = 'active'", userID).Count(&total)
	c.JSON(http.StatusOK, gin.H{"items": novels, "total": total, "limit": limit, "offset": offset})
}

// GetNovel GET /api/novel/:id
func GetNovel(c *gin.Context) {
	userID := c.GetUint64("userID")
	var n db.Novel
	if err := db.DB.First(&n, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "小说不存在"})
		return
	}
	if n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var chapters []db.NovelChapter
	db.DB.Where("novel_id = ?", n.ID).Order("chapter_index ASC").Find(&chapters)
	type chLite struct {
		ID               uint64 `json:"id"`
		ChapterIndex     int    `json:"chapter_index"`
		Title            string `json:"title"`
		StoryboardStatus string `json:"storyboard_status"`
	}
	lite := make([]chLite, 0, len(chapters))
	for _, ch := range chapters {
		lite = append(lite, chLite{ch.ID, ch.ChapterIndex, ch.Title, ch.StoryboardStatus})
	}
	c.JSON(http.StatusOK, gin.H{"novel": n, "chapters": lite})
}

// DeleteNovel DELETE /api/novel/:id
func DeleteNovel(c *gin.Context) {
	userID := c.GetUint64("userID")
	var n db.Novel
	if err := db.DB.First(&n, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "小说不存在"})
		return
	}
	if n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var chIDs []uint64
	db.DB.Model(&db.NovelChapter{}).Where("novel_id = ?", n.ID).Pluck("id", &chIDs)
	if len(chIDs) > 0 {
		db.DB.Where("chapter_id IN ?", chIDs).Delete(&db.NovelShot{})
		db.DB.Where("novel_id = ?", n.ID).Delete(&db.NovelChapter{})
	}
	db.DB.Delete(&n)
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

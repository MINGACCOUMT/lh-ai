package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"google-ai-proxy/internal/config"
	"google-ai-proxy/internal/db"
	"google-ai-proxy/internal/novel"
	"google-ai-proxy/internal/provider"
	"google-ai-proxy/internal/storage"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/simplifiedchinese"
	"gorm.io/gorm"
)

const novelMaxUploadBytes = 50 * 1024 * 1024 // 50MB

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大（上限 50MB）"})
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
	if !utf8.Valid(raw) {
		// 非 UTF-8，按 GBK/GB18030 解码（中文 .txt 常见编码）
		decoded, derr := simplifiedchinese.GB18030.NewDecoder().Bytes(raw)
		if derr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件编码无法识别，请存为 UTF-8 或 GBK 编码"})
			return
		}
		content = string(decoded)
	}

	chapters := novel.ParseChapters(content)
	// 过滤空正文章节（卷首/分节标题如"第一卷 ..."），重排序号
	filtered := chapters[:0]
	for _, ch := range chapters {
		if strings.TrimSpace(ch.Content) != "" {
			filtered = append(filtered, ch)
		}
	}
	for i := range filtered {
		filtered[i].Index = i + 1
	}
	chapters = filtered
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

// GetNovel GET /api/novel/:id  (chapters paginated via ?limit=&offset=)
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
	limit, offset := parseListPagination(c)
	var chapters []db.NovelChapter
	db.DB.Where("novel_id = ?", n.ID).Order("chapter_index ASC").Limit(limit).Offset(offset).Find(&chapters)
	var chapterTotal int64
	db.DB.Model(&db.NovelChapter{}).Where("novel_id = ?", n.ID).Count(&chapterTotal)
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
	c.JSON(http.StatusOK, gin.H{
		"novel":         n,
		"chapters":      lite,
		"chapter_total": chapterTotal,
		"limit":         limit,
		"offset":        offset,
	})
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

// parseUintParam 把路由参数解析为 uint64（若包内已存在同名函数则不要重复定义）。
func parseUintParam(s string) uint64 {
	var n uint64
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			break
		}
		n = n*10 + uint64(ch-'0')
	}
	return n
}

// TriggerStoryboard POST /api/novel/chapters/:id/storyboard
func TriggerStoryboard(c *gin.Context) {
	userID := c.GetUint64("userID")
	chapterID := parseUintParam(c.Param("id"))

	var ch db.NovelChapter
	if err := db.DB.First(&ch, chapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	if ch.StoryboardStatus == "extracting" {
		c.JSON(http.StatusConflict, gin.H{"error": "该章节正在抽取中"})
		return
	}

	credits := config.GetNovelStoryboardCredits()
	if credits > 0 {
		if _, ok := getActiveUser(c, userID); !ok {
			return
		}
		deduct := db.DB.Model(&db.User{}).Where("id = ? AND credits >= ?", userID, credits).
			Update("credits", gorm.Expr("credits - ?", credits))
		if deduct.Error != nil || deduct.RowsAffected == 0 {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "钻石不足"})
			return
		}
		if err := recordCreditTransaction(db.DB, userID, -credits, "novel_storyboard_cost", "novel", "", "分镜抽取"); err != nil {
			refundCredits(userID, credits, "novel-storyboard-ledger-failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "记录流水失败"})
			return
		}
	}

	db.DB.Model(&ch).Update("storyboard_status", "extracting")
	db.DB.Where("chapter_id = ?", chapterID).Delete(&db.NovelShot{})

	go func(chapterID, userID uint64, credits int, content string) {
		status := "ready"
		res, err := novel.ExtractStoryboard(content)
		if err != nil {
			status = "failed"
			log.Printf("[Novel] 分镜抽取失败 [章节:%d]: %v", chapterID, err)
			if credits > 0 {
				refundCredits(userID, credits, "novel-storyboard-failed")
			}
		} else {
			rows := make([]db.NovelShot, 0, len(res.Shots))
			for i, s := range res.Shots {
				rows = append(rows, db.NovelShot{
					ChapterID:  chapterID,
					ShotIndex:  i + 1,
					Scene:      s.Scene,
					Characters: s.Characters,
					Prompt:     s.Prompt,
					Dialogue:   s.Dialogue,
					Camera:     s.Camera,
				})
			}
			if err := db.DB.Create(&rows).Error; err != nil {
				status = "failed"
				log.Printf("[Novel] 分镜保存失败 [章节:%d]: %v", chapterID, err)
				if credits > 0 {
					refundCredits(userID, credits, "novel-storyboard-save-failed")
				}
			} else {
				status = "ready"
				db.DB.Model(&db.NovelChapter{}).Where("id = ?", chapterID).
					Update("outline", res.Outline)
			}
		}
		db.DB.Model(&db.NovelChapter{}).Where("id = ?", chapterID).
			Updates(map[string]interface{}{"storyboard_status": status, "updated_at": time.Now()})
	}(chapterID, userID, credits, ch.Content)

	c.JSON(http.StatusOK, gin.H{"status": "extracting"})
}

// GetStoryboard GET /api/novel/chapters/:id/storyboard
func GetStoryboard(c *gin.Context) {
	userID := c.GetUint64("userID")
	chapterID := parseUintParam(c.Param("id"))
	var ch db.NovelChapter
	if err := db.DB.First(&ch, chapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var shots []db.NovelShot
	db.DB.Where("chapter_id = ?", chapterID).Order("shot_index ASC").Find(&shots)
	c.JSON(http.StatusOK, gin.H{
		"status":     ch.StoryboardStatus,
		"chapter_id": ch.ID,
		"title":      ch.Title,
		"outline":    ch.Outline,
		"content":    ch.Content,
		"shots":      shots,
	})
}

// UpdateShot PUT /api/novel/shots/:id
func UpdateShot(c *gin.Context) {
	userID := c.GetUint64("userID")
	shotID := parseUintParam(c.Param("id"))
	var shot db.NovelShot
	if err := db.DB.First(&shot, shotID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分镜不存在"})
		return
	}
	var ch db.NovelChapter
	if err := db.DB.First(&ch, shot.ChapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var req struct {
		Scene      *string `json:"scene"`
		Characters *string `json:"characters"`
		Prompt     *string `json:"prompt"`
		Dialogue   *string `json:"dialogue"`
		Camera     *string `json:"camera"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}
	updates := map[string]interface{}{}
	if req.Scene != nil {
		updates["scene"] = *req.Scene
	}
	if req.Characters != nil {
		updates["characters"] = *req.Characters
	}
	if req.Prompt != nil {
		updates["prompt"] = *req.Prompt
	}
	if req.Dialogue != nil {
		updates["dialogue"] = *req.Dialogue
	}
	if req.Camera != nil {
		updates["camera"] = *req.Camera
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无更新字段"})
		return
	}
	updates["updated_at"] = time.Now()
	db.DB.Model(&shot).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"message": "已更新"})
}

// AnalyzeChapterHandler POST /api/novel/chapters/:id/analyze
func AnalyzeChapterHandler(c *gin.Context) {
	userID := c.GetUint64("userID")
	chapterID := parseUintParam(c.Param("id"))
	var ch db.NovelChapter
	if err := db.DB.First(&ch, chapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	if strings.TrimSpace(ch.Content) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "本章无正文内容（可能只是卷首/标题），无法解析"})
		return
	}
	if ch.AnalysisStatus == "analyzing" {
		c.JSON(http.StatusConflict, gin.H{"error": "该章节正在解析中"})
		return
	}
	credits := config.GetNovelStoryboardCredits()
	if credits > 0 {
		if _, ok := getActiveUser(c, userID); !ok {
			return
		}
		deduct := db.DB.Model(&db.User{}).Where("id = ? AND credits >= ?", userID, credits).
			Update("credits", gorm.Expr("credits - ?", credits))
		if deduct.Error != nil || deduct.RowsAffected == 0 {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "钻石不足"})
			return
		}
		if err := recordCreditTransaction(db.DB, userID, -credits, "novel_analyze_cost", "novel", "", "章节解析"); err != nil {
			refundCredits(userID, credits, "novel-analyze-ledger-failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "记录流水失败"})
			return
		}
	}
	db.DB.Model(&ch).Update("analysis_status", "analyzing")
	db.DB.Where("chapter_id = ?", chapterID).Delete(&db.NovelPlot{})

	go func(chapterID, userID uint64, credits int, content string) {
		status := "ready"
		res, err := novel.AnalyzeChapter(content)
		if err != nil {
			status = "failed"
			log.Printf("[Novel] 解析失败 [章节:%d]: %v", chapterID, err)
			if credits > 0 {
				refundCredits(userID, credits, "novel-analyze-failed")
			}
		} else {
			plots := make([]db.NovelPlot, 0, len(res.Plots))
			for i, p := range res.Plots {
				plots = append(plots, db.NovelPlot{
					ChapterID: chapterID,
					PlotIndex: i + 1,
					Title:     p.Title,
					Summary:   p.Summary,
				})
			}
			if err := db.DB.Create(&plots).Error; err != nil {
				status = "failed"
				log.Printf("[Novel] 情节保存失败 [章节:%d]: %v", chapterID, err)
				if credits > 0 {
					refundCredits(userID, credits, "novel-analyze-save-failed")
				}
			} else {
				db.DB.Model(&db.NovelChapter{}).Where("id = ?", chapterID).Updates(map[string]interface{}{
					"outline":    res.Outline,
					"characters": res.Characters,
					"scenes":     res.Scenes,
				})
			}
		}
		db.DB.Model(&db.NovelChapter{}).Where("id = ?", chapterID).
			Updates(map[string]interface{}{"analysis_status": status, "updated_at": time.Now()})
	}(chapterID, userID, credits, ch.Content)

	c.JSON(http.StatusOK, gin.H{"status": "analyzing"})
}

// GetChapterDetail GET /api/novel/chapters/:id
func GetChapterDetail(c *gin.Context) {
	userID := c.GetUint64("userID")
	chapterID := parseUintParam(c.Param("id"))
	var ch db.NovelChapter
	if err := db.DB.First(&ch, chapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var plots []db.NovelPlot
	db.DB.Where("chapter_id = ?", chapterID).Order("plot_index ASC").Find(&plots)
	type plotWithShots struct {
		db.NovelPlot
		Shots []db.NovelShot `json:"shots"`
	}
	out := make([]plotWithShots, 0, len(plots))
	for _, p := range plots {
		var shots []db.NovelShot
		db.DB.Where("plot_id = ?", p.ID).Order("shot_index ASC").Find(&shots)
		out = append(out, plotWithShots{NovelPlot: p, Shots: shots})
	}
	c.JSON(http.StatusOK, gin.H{
		"chapter": ch,
		"plots":   out,
	})
}

// StoryboardForPlot POST /api/novel/plots/:id/storyboard
func StoryboardForPlot(c *gin.Context) {
	userID := c.GetUint64("userID")
	plotID := parseUintParam(c.Param("id"))
	var plot db.NovelPlot
	if err := db.DB.First(&plot, plotID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "情节不存在"})
		return
	}
	var ch db.NovelChapter
	if err := db.DB.First(&ch, plot.ChapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	if plot.StoryboardStatus == "extracting" {
		c.JSON(http.StatusConflict, gin.H{"error": "该情节正在生成分镜"})
		return
	}
	credits := config.GetNovelStoryboardCredits()
	if credits > 0 {
		if _, ok := getActiveUser(c, userID); !ok {
			return
		}
		deduct := db.DB.Model(&db.User{}).Where("id = ? AND credits >= ?", userID, credits).
			Update("credits", gorm.Expr("credits - ?", credits))
		if deduct.Error != nil || deduct.RowsAffected == 0 {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "钻石不足"})
			return
		}
		if err := recordCreditTransaction(db.DB, userID, -credits, "novel_plot_storyboard_cost", "novel", "", "情节分镜"); err != nil {
			refundCredits(userID, credits, "novel-plot-storyboard-ledger-failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "记录流水失败"})
			return
		}
	}
	db.DB.Model(&plot).Update("storyboard_status", "extracting")
	db.DB.Where("plot_id = ?", plotID).Delete(&db.NovelShot{})

	go func(plotID, userID uint64, credits int, p db.NovelPlot, chChars, chScenes string) {
		status := "ready"
		shots, err := novel.ExtractShotsForPlot(novel.Plot{Title: p.Title, Summary: p.Summary}, chChars, chScenes)
		if err != nil {
			status = "failed"
			log.Printf("[Novel] 情节分镜失败 [情节:%d]: %v", plotID, err)
			if credits > 0 {
				refundCredits(userID, credits, "novel-plot-storyboard-failed")
			}
		} else {
			rows := make([]db.NovelShot, 0, len(shots))
			for i, s := range shots {
				rows = append(rows, db.NovelShot{
					PlotID:     plotID,
					ShotIndex:  i + 1,
					Scene:      s.Scene,
					Characters: s.Characters,
					Prompt:     s.Prompt,
					Dialogue:   s.Dialogue,
					Camera:     s.Camera,
				})
			}
			if err := db.DB.Create(&rows).Error; err != nil {
				status = "failed"
				log.Printf("[Novel] 情节分镜保存失败 [情节:%d]: %v", plotID, err)
				if credits > 0 {
					refundCredits(userID, credits, "novel-plot-storyboard-save-failed")
				}
			}
		}
		db.DB.Model(&db.NovelPlot{}).Where("id = ?", plotID).
			Update("storyboard_status", status)
	}(plotID, userID, credits, plot, ch.Characters, ch.Scenes)

	c.JSON(http.StatusOK, gin.H{"status": "extracting"})
}

// GenerateAssets POST /api/novel/chapters/:id/assets
// 解析本章人物画像/场景文本，为每个新角色/场景生成图像，存入小说级资产库。
func GenerateAssets(c *gin.Context) {
	userID := c.GetUint64("userID")
	chapterID := parseUintParam(c.Param("id"))
	var ch db.NovelChapter
	if err := db.DB.First(&ch, chapterID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "章节不存在"})
		return
	}
	var n db.Novel
	if err := db.DB.First(&n, ch.NovelID).Error; err != nil || n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	if ch.AssetsStatus == "generating" {
		c.JSON(http.StatusConflict, gin.H{"error": "该章节正在生成资产"})
		return
	}
	if ch.AnalysisStatus != "ready" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先解析章节"})
		return
	}

	db.DB.Model(&ch).Update("assets_status", "generating")

	go func(chapterID, novelID, userID uint64, charText, sceneText string) {
		status := "ready"
		userIDStr := strconv.FormatUint(userID, 10)
		_ = userIDStr // reserved for future per-user asset path/namespacing
		// 生成角色图
		for _, entry := range parseAssetEntries(charText) {
			var existing db.NovelCharacter
			if db.DB.Where("novel_id = ? AND name = ?", novelID, entry.Name).First(&existing).Error == nil {
				continue // 已存在，复用
			}
			imgURL, err := generateAssetImage(entry.Name, entry.Description, "角色立绘")
			if err != nil {
				log.Printf("[Novel] 角色图生成失败 [%s]: %v", entry.Name, err)
				continue
			}
			db.DB.Create(&db.NovelCharacter{
				NovelID: novelID, Name: entry.Name, Description: entry.Description, ImageURL: imgURL,
			})
		}
		// 生成场景图
		for _, entry := range parseAssetEntries(sceneText) {
			var existing db.NovelScene
			if db.DB.Where("novel_id = ? AND name = ?", novelID, entry.Name).First(&existing).Error == nil {
				continue
			}
			imgURL, err := generateAssetImage(entry.Name, entry.Description, "场景概念图")
			if err != nil {
				log.Printf("[Novel] 场景图生成失败 [%s]: %v", entry.Name, err)
				continue
			}
			db.DB.Create(&db.NovelScene{
				NovelID: novelID, Name: entry.Name, Description: entry.Description, ImageURL: imgURL,
			})
		}
		db.DB.Model(&db.NovelChapter{}).Where("id = ?", chapterID).Update("assets_status", status)
	}(chapterID, n.ID, userID, ch.Characters, ch.Scenes)

	c.JSON(http.StatusOK, gin.H{"status": "generating"})
}

// GetAssets GET /api/novel/:id/assets
func GetAssets(c *gin.Context) {
	userID := c.GetUint64("userID")
	novelID := parseUintParam(c.Param("id"))
	var n db.Novel
	if err := db.DB.First(&n, novelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "小说不存在"})
		return
	}
	if n.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问"})
		return
	}
	var characters []db.NovelCharacter
	db.DB.Where("novel_id = ?", novelID).Order("created_at ASC").Find(&characters)
	var scenes []db.NovelScene
	db.DB.Where("novel_id = ?", novelID).Order("created_at ASC").Find(&scenes)
	c.JSON(http.StatusOK, gin.H{"characters": characters, "scenes": scenes})
}

// parseAssetEntries 把 "name：description\nname：description" 文本解析成条目列表。
// 优先按中文全角 "：" 分割，回退到 ASCII ":"。
func parseAssetEntries(text string) []struct{ Name, Description string } {
	var entries []struct{ Name, Description string }
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// 优先按全角 "："（UTF-8 3 字节）分割
		idx := strings.Index(line, "：")
		sepLen := len("：")
		if idx < 0 {
			// 回退到 ASCII ":"
			idx = strings.Index(line, ":")
			sepLen = 1
		}
		if idx > 0 {
			entries = append(entries, struct{ Name, Description string }{
				Name:        strings.TrimSpace(line[:idx]),
				Description: strings.TrimSpace(line[idx+sepLen:]),
			})
		} else {
			entries = append(entries, struct{ Name, Description string }{Name: line, Description: ""})
		}
	}
	return entries
}

// generateAssetImage 调图像模型生成一张资产图，上传 OSS，返回公开 URL。
func generateAssetImage(name, description, kind string) (string, error) {
	gen, err := provider.Get("gpt-image-2")
	if err != nil {
		return "", fmt.Errorf("图像模型不可用: %v", err)
	}
	prompt := fmt.Sprintf("%s：%s。%s。高质量，细节丰富，电影级光影，4K", kind, name, description)
	result, err := gen.GenerateImage(prompt, provider.ImageOptions{AspectRatio: "1:1", ImageSize: "1K"})
	if err != nil {
		return "", err
	}
	// UploadBase64Image(base64Data, licenseID, directory)
	return storage.UploadBase64Image(result.Data, "novel-assets", "novel-assets")
}

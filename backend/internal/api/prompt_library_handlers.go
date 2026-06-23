package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// PromptEntry mirrors one record of docs/awesome-prompt-main/.../data/prompts-all.json
type PromptEntry struct {
	ID        string `json:"id"`
	Image     string `json:"image"`
	MediaType string `json:"mediaType"`
	Title     string `json:"title"`
	Prompt    string `json:"prompt"`
}

var (
	promptLibraryOnce sync.Once
	promptLibrary     []PromptEntry
	promptLibraryErr  error
)

// loadPromptLibrary lazily parses the prompts-all.json catalog exactly once.
// The 9.8MB file is kept in memory after the first request.
func loadPromptLibrary() ([]PromptEntry, error) {
	promptLibraryOnce.Do(func() {
		// Candidate paths cover running from repo root, backend/, or other layouts.
		candidates := []string{
			filepath.Join("docs", "awesome-prompt-main", "awesome-prompt-main", "data", "prompts-all.json"),
			filepath.Join("..", "docs", "awesome-prompt-main", "awesome-prompt-main", "data", "prompts-all.json"),
			filepath.Join("..", "..", "docs", "awesome-prompt-main", "awesome-prompt-main", "data", "prompts-all.json"),
		}
		var data []byte
		var err error
		for _, p := range candidates {
			if data, err = os.ReadFile(p); err == nil {
				log.Printf("[PromptLibrary] loaded from %s", p)
				break
			}
		}
		if err != nil {
			promptLibraryErr = fmt.Errorf("prompt library not found: %v", err)
			return
		}
		if err := json.Unmarshal(data, &promptLibrary); err != nil {
			promptLibraryErr = fmt.Errorf("prompt library parse error: %v", err)
			return
		}
		log.Printf("[PromptLibrary] parsed %d entries", len(promptLibrary))
	})
	return promptLibrary, promptLibraryErr
}

// parsePromptPagination allows larger pages than the inspirations helper (up to 200).
func parsePromptPagination(c *gin.Context) (limit int, offset int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	return
}

// ListPrompts GET /api/prompts?type=image|video&all=1&q=keyword&limit=&offset=
//
// Public endpoint (no auth) serving the Awesome Prompt catalog.
//   - type: filter by mediaType ("image" | "video"); omit / "all" returns both
//   - q:    case-insensitive substring search over title and prompt
//   - limit/offset: pagination (default 20, max 200)
func ListPrompts(c *gin.Context) {
	entries, err := loadPromptLibrary()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "提示词库不可用: " + err.Error()})
		return
	}

	mediaType := strings.TrimSpace(c.Query("type"))
	q := strings.ToLower(strings.TrimSpace(c.Query("q")))

	// Pre-filter by type only when no search is requested — this avoids allocating
	// a full-size filtered slice on the common (unfiltered) path.
	if mediaType == "" || mediaType == "all" {
		if q == "" {
			// No filtering at all: paginate directly over the source slice.
			limit, offset := parsePromptPagination(c)
			total := len(entries)
			end := offset + limit
			if end > total {
				end = total
			}
			if offset > total {
				offset = total
			}
			c.JSON(http.StatusOK, gin.H{
				"items":  entries[offset:end],
				"total":  total,
				"limit":  limit,
				"offset": offset,
			})
			return
		}
		mediaType = "" // search across all types
	}

	filtered := make([]PromptEntry, 0, len(entries))
	for _, e := range entries {
		if mediaType != "" && e.MediaType != mediaType {
			continue
		}
		if q != "" {
			if !(strings.Contains(strings.ToLower(e.Title), q) ||
				strings.Contains(strings.ToLower(e.Prompt), q)) {
				continue
			}
		}
		filtered = append(filtered, e)
	}

	limit, offset := parsePromptPagination(c)
	total := len(filtered)
	end := offset + limit
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	items := filtered[offset:end]

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

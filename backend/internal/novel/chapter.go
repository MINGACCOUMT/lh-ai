package novel

import (
	"regexp"
	"strings"
)

// chapterTitleRe 匹配行首的章节标题：第一章 / 第123章 / 第二节 / 第一回 ...
// 注意：用 [ \t]* 而非 [[:space:]]*——后者含 \n，会把标题前的空行换行吃进匹配起点，
// 导致标题段以 \n 开头、标题解析为空（章节间有空行时复现）。
var chapterTitleRe = regexp.MustCompile(`(?m)^[ \t]*第[0-9一二三四五六七八九十百千零〇两]+[章节回卷部篇][^.\n]*$`)

// ParsedChapter 切出来的章节。
type ParsedChapter struct {
	Index   int
	Title   string
	Content string
}

// ParseChapters 用正则把小说正文切成章节。未匹配到任何标题 → 整文作为 1 章（title=全文）。
func ParseChapters(raw string) []ParsedChapter {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	locs := chapterTitleRe.FindAllStringIndex(raw, -1)
	if len(locs) == 0 {
		return []ParsedChapter{{Index: 1, Title: "全文", Content: strings.TrimSpace(raw)}}
	}
	out := make([]ParsedChapter, 0, len(locs))
	for i, loc := range locs {
		start := loc[0]
		end := len(raw)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		seg := raw[start:end]
		title := seg
		body := ""
		if nl := strings.IndexByte(seg, '\n'); nl >= 0 {
			title = seg[:nl]
			body = seg[nl+1:]
		}
		out = append(out, ParsedChapter{
			Index:   i + 1,
			Title:   strings.TrimSpace(title),
			Content: strings.TrimSpace(body),
		})
	}
	return out
}

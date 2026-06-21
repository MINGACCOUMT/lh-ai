package novel

import "testing"

func TestParseChapters_MultiChapter(t *testing.T) {
	raw := "第一章 初遇\n小明遇见了小红。\n第二章 重逢\n十年后他们再次相遇。"
	cs := ParseChapters(raw)
	if len(cs) != 2 {
		t.Fatalf("want 2 chapters, got %d", len(cs))
	}
	if cs[0].Title != "第一章 初遇" {
		t.Errorf("title0=%q", cs[0].Title)
	}
	if cs[0].Content != "小明遇见了小红。" {
		t.Errorf("content0=%q", cs[0].Content)
	}
	if cs[1].Index != 2 || cs[1].Title != "第二章 重逢" {
		t.Errorf("chapter2 wrong: %+v", cs[1])
	}
}

func TestParseChapters_NoMarker_WholeAsOne(t *testing.T) {
	raw := "这是一段没有章节标题的文字。继续。"
	cs := ParseChapters(raw)
	if len(cs) != 1 {
		t.Fatalf("want 1 chapter, got %d", len(cs))
	}
	if cs[0].Title != "全文" {
		t.Errorf("title=%q want 全文", cs[0].Title)
	}
	if cs[0].Content != raw {
		t.Errorf("content should be whole text")
	}
}

func TestParseChapters_ArabicNumerals(t *testing.T) {
	raw := "第1章 开端\n内容A\n第2章 发展\n内容B"
	cs := ParseChapters(raw)
	if len(cs) != 2 || cs[0].Title != "第1章 开端" || cs[1].Title != "第2章 发展" {
		t.Errorf("arabic numeral parse wrong: %+v", cs)
	}
}

func TestParseChapters_Hui(t *testing.T) {
	raw := "第一回 风雪夜\n内容。\n第二回 归途\n内容2。"
	cs := ParseChapters(raw)
	if len(cs) != 2 {
		t.Fatalf("want 2 (回), got %d", len(cs))
	}
}

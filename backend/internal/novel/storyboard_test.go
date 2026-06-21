package novel

import "testing"

func TestParseStoryboardJSON_Valid(t *testing.T) {
	raw := `{"outline":"本章讲初遇。","shots":[{"scene":"教室","characters":"小明,老师","prompt":"教室阳光","dialogue":"你好","camera":"推进"},{"scene":"操场","characters":"小明","prompt":"操场奔跑","dialogue":"","camera":"平移"}]}`
	res, err := parseStoryboardJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Outline != "本章讲初遇。" {
		t.Errorf("outline=%q", res.Outline)
	}
	if len(res.Shots) != 2 || res.Shots[0].Scene != "教室" {
		t.Errorf("shots wrong: %+v", res.Shots)
	}
}

func TestParseStoryboardJSON_TruncateTo9(t *testing.T) {
	raw := `{"outline":"o","shots":[`
	for i := 0; i < 12; i++ {
		if i > 0 {
			raw += ","
		}
		raw += `{"scene":"s` + itoa(i) + `","prompt":"p"}`
	}
	raw += `]}`
	res, err := parseStoryboardJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(res.Shots) != 9 {
		t.Errorf("want 9, got %d", len(res.Shots))
	}
}

func TestParseStoryboardJSON_CodeFence(t *testing.T) {
	raw := "```json\n{\"outline\":\"x\",\"shots\":[{\"scene\":\"y\",\"prompt\":\"z\"}]}\n```"
	res, err := parseStoryboardJSON(raw)
	if err != nil || len(res.Shots) != 1 {
		t.Errorf("code-fence failed: %v, %d", err, len(res.Shots))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

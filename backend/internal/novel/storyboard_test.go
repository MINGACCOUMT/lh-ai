package novel

import "testing"

func TestParseShotsJSON_Valid(t *testing.T) {
	raw := `{"shots":[{"scene":"教室","characters":"小明,老师","prompt":"教室阳光","dialogue":"你好","camera":"推进"},{"scene":"操场","characters":"小明","prompt":"操场奔跑","dialogue":"","camera":"平移"}]}`
	shots, err := parseShotsJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(shots) != 2 {
		t.Fatalf("want 2, got %d", len(shots))
	}
	if shots[0].Scene != "教室" || shots[0].Prompt != "教室阳光" {
		t.Errorf("shot0 wrong: %+v", shots[0])
	}
}

func TestParseShotsJSON_TruncateTo9(t *testing.T) {
	raw := `{"shots":[`
	for i := 0; i < 12; i++ {
		if i > 0 {
			raw += ","
		}
		raw += `{"scene":"s` + itoa(i) + `","prompt":"p"}`
	}
	raw += `]}`
	shots, err := parseShotsJSON(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(shots) != 9 {
		t.Errorf("want 9 (truncated), got %d", len(shots))
	}
}

func TestParseShotsJSON_CodeFence(t *testing.T) {
	raw := "```json\n{\"shots\":[{\"scene\":\"x\",\"prompt\":\"y\"}]}\n```"
	shots, err := parseShotsJSON(raw)
	if err != nil || len(shots) != 1 {
		t.Errorf("code-fence parse failed: %v, %d", err, len(shots))
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

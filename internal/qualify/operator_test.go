package qualify

import (
	"strings"
	"testing"
)

func TestParseOperatorAnswer(t *testing.T) {
	pass, ok := ParseOperatorAnswer(" PASS \n")
	if !ok || pass != ResultPass {
		t.Fatalf("pass %v %v", pass, ok)
	}
	fail, ok := ParseOperatorAnswer("fail")
	if !ok || fail != ResultFail {
		t.Fatalf("fail")
	}
	skip, ok := ParseOperatorAnswer("s")
	if !ok || skip != ResultSkip {
		t.Fatalf("skip")
	}
	if _, ok := ParseOperatorAnswer("1234567890121"); ok {
		t.Fatal("citizen id must be refused")
	}
	if _, ok := ParseOperatorAnswer("data:image/jpeg;base64,QQ=="); ok {
		t.Fatal("photo must be refused")
	}
	if _, ok := ParseOperatorAnswer("นายสมชาย ใจดี"); ok {
		t.Fatal("name must be refused")
	}
	if _, ok := ParseOperatorAnswer("maybe later with a long reason"); ok {
		t.Fatal("free text must be refused")
	}
	if _, ok := ParseOperatorAnswer(""); ok {
		t.Fatal("empty")
	}
}

func TestOperatorPromptsHaveNoPII(t *testing.T) {
	seen := map[string]bool{}
	for _, step := range OperatorChecks() {
		seen[step.ID] = true
		blob := step.ID + step.Name + step.Prompt + step.Detail
		if hit := FirstForbidden([]byte(blob)); hit != "" {
			t.Fatalf("%s leaked %q", step.ID, hit)
		}
		if !strings.Contains(step.Prompt, "pass, fail, or skip") {
			t.Fatalf("%s missing answer hint", step.ID)
		}
	}
	for _, id := range []string{"E4", "E5", "E6", "E7", "M1"} {
		if !seen[id] {
			t.Fatalf("missing %s", id)
		}
	}
	e4 := OperatorChecks()[0]
	if e4.ID != "E4" {
		t.Fatalf("first step %s", e4.ID)
	}
	for _, token := range []string{"CARD_REMOVED", "READER_REMOVED", "READ_FAILED", "พร้อม"} {
		if !strings.Contains(e4.Prompt, token) && !strings.Contains(e4.Detail, token) {
			t.Fatalf("E4 mapping missing %s", token)
		}
	}
}

func TestFirstForbiddenJPEGBytesAndPNG(t *testing.T) {
	if FirstForbidden([]byte{0x00, 0xFF, 0xD8, 0xFF, 0x01}) != "jpeg-soi" {
		t.Fatal("jpeg soi")
	}
	if FirstForbidden([]byte("iVBORw0KGgoAAA")) == "" {
		t.Fatal("png")
	}
	if FirstForbidden([]byte("FFD8FFE0")) == "" {
		t.Fatal("jpeg hex")
	}
}

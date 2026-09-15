package card

import (
	"strings"
	"testing"
)

func TestNormalizeFields(t *testing.T) {
	got, err := NormalizeFields([]Field{FieldThaiName, FieldThaiName, FieldBirthDate})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != FieldThaiName || got[1] != FieldBirthDate {
		t.Fatalf("got %v", got)
	}
	if _, err := NormalizeFields(nil); err == nil {
		t.Fatal("expected empty list error")
	}
	if _, err := NormalizeFields([]Field{"portrait"}); err == nil {
		t.Fatal("expected unsupported field error")
	}
}

func TestSubset(t *testing.T) {
	if !Subset([]Field{FieldCitizenID}, DefaultFields) {
		t.Fatal("expected subset")
	}
	if Subset([]Field{"portrait"}, DefaultFields) {
		t.Fatal("unexpected subset")
	}
}

func TestRetryableCodes(t *testing.T) {
	if !Retryable(ErrBusy) || !Retryable(ErrReadTimeout) {
		t.Fatal("busy/timeout should be retryable")
	}
	if Retryable(ErrApprovalDenied) || Retryable(ErrUnsupportedCard) {
		t.Fatal("denial/unsupported should not be retryable")
	}
}

func TestSW(t *testing.T) {
	sw, ok := SW([]byte{0x01, 0x90, 0x00})
	if !ok || sw != StatusOK {
		t.Fatalf("got %x %v", sw, ok)
	}
	if _, ok := SW([]byte{0x90}); ok {
		t.Fatal("short body should fail")
	}
}

func TestErrorMessageHasNoFieldContents(t *testing.T) {
	e := NewError(ErrReadFailed, "read failed")
	if strings.Contains(e.Error(), "110170") {
		t.Fatal("error message leaked sentinel")
	}
}

package qualify

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/night-sornram/openthaiid/internal/card"
)

func TestSummarizeReadOmitsPersonalValues(t *testing.T) {
	male := card.GenderMale
	cid := "1234567890121"
	title := "นาย"
	given := "สมชาย"
	family := "ใจดี"
	enTitle := "Mr."
	enGiven := "Somchai"
	enFamily := "Jaidee"
	issuer := "เทศบาลนครนิทาน"
	house := "99"
	street := "ถนนนิทาน"
	province := "จังหวัดนิทาน"
	y, m, d := 1990, 4, 12
	jpeg := card.FictionalPortraitJPEG()
	res := &card.ReadResult{
		Fields: card.FieldValues{
			CitizenID: &cid,
			ThaiName: &card.ThaiName{
				Title: &title, GivenName: &given, FamilyName: &family,
				FullName: "นาย สมชาย ใจดี",
			},
			EnglishName: &card.ThaiName{
				Title: &enTitle, GivenName: &enGiven, FamilyName: &enFamily,
				FullName: "Mr. Somchai Jaidee",
			},
			BirthDate: &card.CardDate{Year: &y, Month: &m, Day: &d},
			Gender:    &card.Gender{Code: "1", Value: &male},
			Issuer:    &issuer,
			Address: &card.ThaiAddress{
				HouseNo: &house, Street: &street, Province: &province,
				FullAddress: "99 ถนนนิทาน จังหวัดนิทาน",
			},
			Portrait: &card.Portrait{
				MimeType: card.PortraitMimeJPEG,
				Encoding: card.PortraitEncodingBase64,
				Data:     base64.StdEncoding.EncodeToString(jpeg),
			},
		},
	}
	shape := SummarizeRead(res)
	raw, err := json.Marshal(shape)
	if err != nil {
		t.Fatal(err)
	}
	if hit := FirstForbidden(raw); hit != "" {
		t.Fatalf("shape leaked %q: %s", hit, raw)
	}
	if !shape.CitizenID.Present || shape.CitizenID.DigitCount != 13 {
		t.Fatalf("citizenId shape %+v", shape.CitizenID)
	}
	if shape.Gender.Code != "1" || shape.Gender.Value != "male" || !shape.Gender.CodeDigit {
		t.Fatalf("gender %+v", shape.Gender)
	}
	if !shape.Portrait.Present || !shape.Portrait.JPEGSOI || !shape.Portrait.JPEGEOI {
		t.Fatalf("portrait %+v", shape.Portrait)
	}
	if shape.Address.ComponentCount < 3 {
		t.Fatalf("address %+v", shape.Address)
	}
	joined := strings.Join(shape.ThaiName.PartsPresent, ",")
	if !strings.Contains(joined, "givenName") || !strings.Contains(joined, "familyName") {
		t.Fatalf("thai name parts %v", shape.ThaiName.PartsPresent)
	}
}

func TestFirstForbiddenCatchesCitizenIDAndJPEG(t *testing.T) {
	if FirstForbidden([]byte(`{"ok":true}`)) != "" {
		t.Fatal("clean json flagged")
	}
	if FirstForbidden([]byte("cid 1234567890121")) == "" {
		t.Fatal("expected citizen id")
	}
	if FirstForbidden([]byte(card.FictionalPortraitSentinel)) == "" {
		t.Fatal("expected portrait sentinel")
	}
	if FirstForbidden([]byte("data:image/jpeg;base64,QQ==")) == "" {
		t.Fatal("expected data url")
	}
}

func TestMarshalSafeBlocksPoisonedReport(t *testing.T) {
	r := NewReport("test", "synthetic-smoke")
	r.AddCheck(Check{ID: "E1", Name: "x", Result: ResultPass, Detail: "reader+card visible in 12ms"})
	r.Finalize()
	if _, err := r.MarshalSafe(); err != nil {
		t.Fatal(err)
	}
	r.Checks[0].Detail = "patient 1234567890121"
	if _, err := r.MarshalSafe(); err == nil {
		t.Fatal("expected block")
	}
}

func TestThirteenDigitTokenIgnoresDigitCount(t *testing.T) {
	if FirstForbidden([]byte(`{"digitCount":13,"passed":100}`)) != "" {
		t.Fatal("digitCount:13 should not look like a citizen ID")
	}
}

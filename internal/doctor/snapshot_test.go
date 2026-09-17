package doctor

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/qualify"
)

func TestExitCodeHardFailOnly(t *testing.T) {
	cases := []struct {
		name string
		snap Snapshot
		want int
	}{
		{
			name: "ready desk",
			snap: Snapshot{
				Version:       "0.1.0-rc.8",
				Helper:        HelperRunning,
				HTTP:          "127.0.0.1:17891",
				TLS:           TLSNo,
				ReadyURL:      "http://127.0.0.1:17891/",
				PCSC:          PCSCOK,
				Backend:       "pcsc",
				BackendStatus: "available",
				Readers:       []Reader{{Name: "ACS ACR39U", Presence: "present", Selected: true}},
				Probe:         Probe{OK: true, Status: "ok", Backend: "available"},
				Update:        Update{State: UpdateUnavailable, Reason: "offline", Soft: true},
			},
			want: 0,
		},
		{
			name: "helper down",
			snap: Snapshot{Helper: HelperDown, PCSC: PCSCOK, Readers: []Reader{{Name: "ACS ACR39U"}}},
			want: 1,
		},
		{
			name: "pcsc unavailable",
			snap: Snapshot{Helper: HelperRunning, PCSC: PCSCFail, BackendStatus: "unavailable"},
			want: 1,
		},
		{
			name: "no reader is soft",
			snap: Snapshot{Helper: HelperRunning, PCSC: PCSCOK, BackendStatus: "available"},
			want: 0,
		},
		{
			name: "soft update failure",
			snap: Snapshot{
				Helper:        HelperRunning,
				PCSC:          PCSCOK,
				BackendStatus: "available",
				Readers:       []Reader{{Name: "SYNTHETIC ACR39U (fictional demo)", Presence: "present", Selected: true}},
				Update:        Update{State: UpdateUnavailable, Reason: "rate_limited", Soft: true},
			},
			want: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExitCode(Finalize(tc.snap))
			if got != tc.want {
				t.Fatalf("exit %d want %d", got, tc.want)
			}
		})
	}
}

func TestFormatRedactsPIIAndSecrets(t *testing.T) {
	s := Snapshot{
		Version:       "0.1.0-rc.8",
		Helper:        HelperRunning,
		HTTP:          "127.0.0.1:17891",
		TLS:           TLSYes,
		TLSListen:     "/home/tech/loopback-key.pem",
		ReadyURL:      "http://127.0.0.1:17891/",
		PCSC:          PCSCOK,
		Backend:       "pcsc",
		BackendStatus: "available",
		Readers: []Reader{
			{Name: "สมชาย ใจดี 1234567890121", Presence: "present", Selected: true},
		},
		Probe:  Probe{OK: true, Backend: "available"},
		Update: Update{State: UpdateNotChecked, Soft: true, Tag: "github_pat_abcdefghijklmnopqrstu"},
		Hint:   "tlsKeyFile=/secret/loopback-key.pem -----BEGIN PRIVATE KEY-----",
		NextTH: "Somchai Jaidee",
		NextEN: "tok_abcdef",
	}
	out := Format(s)
	if hit := qualify.FirstForbidden([]byte(out)); hit != "" {
		t.Fatalf("format leaked %q: %s", hit, out)
	}
	for _, n := range []string{
		"สมชาย", "1234567890121", "Somchai", "loopback-key.pem",
		"BEGIN PRIVATE KEY", "github_pat_", "tlsKeyFile", "tok_abcdef",
	} {
		if strings.Contains(out, n) {
			t.Fatalf("format leaked %q: %s", n, out)
		}
	}
	if !strings.Contains(out, "personal fields: not read") {
		t.Fatalf("missing personal-fields line: %s", out)
	}
	if strings.Contains(out, "tls=yes") && strings.Contains(out, ".pem") {
		t.Fatalf("tls yes must not include cert/key paths: %s", out)
	}
}

func TestFormatHealthyKeepsKitMarkers(t *testing.T) {
	out := Format(Snapshot{
		Version:       "0.1.0-rc.8",
		Helper:        HelperRunning,
		HTTP:          "127.0.0.1:17891",
		TLS:           TLSNo,
		ReadyURL:      "http://127.0.0.1:17891/",
		PCSC:          PCSCOK,
		Backend:       "pcsc",
		BackendStatus: "available",
		Readers:       []Reader{{Name: "ACS ACR39U ICC Reader", Presence: "present", Selected: true}},
		Probe:         Probe{OK: true, Backend: "available"},
		Update:        Update{State: UpdateNotChecked, Soft: true},
	})
	for _, want := range []string{
		"openthaiid doctor (0.1.0-rc.8)",
		"result: OK",
		"pcsc: OK",
		"ACS ACR39U ICC Reader",
		"personal fields: not read",
		"tls=no",
		"probe: OK backend=available",
		"ถัดไป:",
		"Next:",
		"http://127.0.0.1:17891/",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if ContainsForbidden(out) {
		t.Fatalf("healthy output flagged forbidden: %s", out)
	}
}

func TestCollectHelperDownUsesIndependentPCSC(t *testing.T) {
	cfg := config.Default()
	cfg.TLSCertFile = "/etc/openthaiid/loopback.pem"
	cfg.TLSKeyFile = "/etc/openthaiid/loopback-key.pem"
	cfg.TLSPort = 17892
	snap := Collect(context.Background(), Options{
		Version: "0.1.0-rc.8",
		Config:  cfg,
		Helper: func(context.Context) (*Snapshot, error) {
			return nil, errors.New("connection refused")
		},
		PCSC: func(context.Context) (Inventory, error) {
			return Inventory{
				Backend: "pcsc",
				Readers: []Reader{{Name: "ACS ACR39U", Presence: "absent"}},
			}, nil
		},
	})
	if ExitCode(snap) != 1 {
		t.Fatalf("helper down must be hard fail: %+v", snap)
	}
	if snap.FailCode != FailHelperDown {
		t.Fatalf("fail %q", snap.FailCode)
	}
	if snap.PCSC != PCSCOK || len(snap.Readers) != 1 {
		t.Fatalf("independent pcsc %+v", snap)
	}
	if snap.TLS != TLSYes {
		t.Fatalf("tls from config %+v", snap)
	}
	out := Format(snap)
	if strings.Contains(out, "loopback-key.pem") || strings.Contains(out, "TLSKey") {
		t.Fatalf("key path leaked: %s", out)
	}
	if !strings.Contains(out, "tls=yes") || !strings.Contains(out, "https=127.0.0.1:17892") {
		t.Fatalf("expected tls yes + port: %s", out)
	}
	if !strings.Contains(out, "probe: skipped (helper down)") {
		t.Fatalf("probe line: %s", out)
	}
	if !strings.Contains(out, "เปิด OpenThaiID") || !strings.Contains(out, "Start OpenThaiID") {
		t.Fatalf("next step: %s", out)
	}
}

func TestCollectHelperUpDoesNotOpenPCSC(t *testing.T) {
	var listed bool
	snap := Collect(context.Background(), Options{
		Version: "cli",
		Config:  config.Default(),
		Helper: func(context.Context) (*Snapshot, error) {
			s := Snapshot{
				Version:       "0.1.0-rc.8",
				Helper:        HelperRunning,
				HTTP:          "127.0.0.1:17991",
				TLS:           TLSNo,
				ReadyURL:      "http://127.0.0.1:17991/",
				PCSC:          PCSCOK,
				Backend:       "synthetic",
				BackendStatus: "available",
				Practice:      true,
				Readers:       []Reader{{Name: "SYNTHETIC ACR39U (fictional demo)", Presence: "present", Selected: true}},
				Probe:         Probe{OK: true, Backend: "available"},
				Update:        Update{State: UpdateUnavailable, Reason: "offline", Soft: true},
			}
			return &s, nil
		},
		PCSC: func(context.Context) (Inventory, error) {
			listed = true
			return Inventory{}, errors.New("must not list pcsc while helper is up")
		},
	})
	if listed {
		t.Fatal("running helper must not open a second PC/SC context")
	}
	if ExitCode(snap) != 0 {
		t.Fatalf("exit %d snap=%+v", ExitCode(snap), snap)
	}
	out := Format(snap)
	if !strings.Contains(out, "probe: OK") || !strings.Contains(out, "unavailable reason=offline") {
		t.Fatalf("format: %s", out)
	}
	if !strings.Contains(out, "personal fields: not read") {
		t.Fatal(out)
	}
	if qualify.ContainsForbidden(out) {
		t.Fatalf("leaked: %s", out)
	}
}

func TestCollectPCSCFailWhileHelperUp(t *testing.T) {
	snap := Collect(context.Background(), Options{
		Version: "0.1.0-rc.8",
		Helper: func(context.Context) (*Snapshot, error) {
			s := Snapshot{
				Version:       "0.1.0-rc.8",
				Helper:        HelperRunning,
				HTTP:          "127.0.0.1:17891",
				TLS:           TLSNo,
				ReadyURL:      "http://127.0.0.1:17891/",
				PCSC:          PCSCFail,
				Backend:       "pcsc",
				BackendStatus: "unavailable",
				Probe:         Probe{OK: true, Backend: "unavailable"},
				Update:        Update{State: UpdateNotChecked, Soft: true},
			}
			return &s, nil
		},
		PCSC: func(context.Context) (Inventory, error) {
			t.Fatal("must not list pcsc")
			return Inventory{}, nil
		},
	})
	if ExitCode(snap) != 1 || snap.FailCode != FailPCSC {
		t.Fatalf("%+v", snap)
	}
	out := Format(snap)
	if strings.Contains(out, "pcsc: OK") {
		t.Fatalf("must not claim pcsc OK: %s", out)
	}
	if !strings.Contains(out, "pcsc: FAIL") {
		t.Fatalf("want FAIL: %s", out)
	}
	if !strings.Contains(out, "Reconnect card service") && !strings.Contains(out, "ทดสอบการเชื่อมต่อ") {
		t.Fatalf("next: %s", out)
	}
}

func TestCollectHelperDownPCSCFail(t *testing.T) {
	snap := Collect(context.Background(), Options{
		Version: "0.1.0-rc.8",
		Config:  config.Default(),
		Helper:  func(context.Context) (*Snapshot, error) { return nil, errors.New("down") },
		PCSC:    func(context.Context) (Inventory, error) { return Inventory{}, errors.New("no pcscd") },
	})
	if ExitCode(snap) != 1 || snap.FailCode != FailHelperDown {
		t.Fatalf("%+v", snap)
	}
	if snap.PCSC != PCSCFail {
		t.Fatalf("pcsc %+v", snap)
	}
	out := Format(snap)
	if !strings.Contains(out, "pcscd") && !strings.Contains(out, "Smart Card") {
		t.Fatalf("hint: %s", out)
	}
}

func TestFormatBilingualNextAndNoCardFields(t *testing.T) {
	out := Format(Snapshot{
		Helper:        HelperRunning,
		PCSC:          PCSCOK,
		BackendStatus: "available",
		HTTP:          "127.0.0.1:17891",
		TLS:           TLSNo,
		ReadyURL:      "http://127.0.0.1:17891/",
		Readers: []Reader{
			{Name: "Reader One", Presence: "present"},
			{Name: "Reader Two", Presence: "present"},
		},
	})
	if !strings.Contains(out, "ถัดไป:") || !strings.Contains(out, "Next:") {
		t.Fatalf("bilingual next missing: %s", out)
	}
	if !strings.Contains(out, "เลือกเครื่อง") || !strings.Contains(out, "More than one reader") {
		t.Fatalf("choose-reader next: %s", out)
	}
	for _, n := range []string{"citizenId", "thaiName", "portrait", "token"} {
		if strings.Contains(out, n) {
			t.Fatalf("unexpected %q: %s", n, out)
		}
	}
}

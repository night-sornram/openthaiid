package bridge

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/night-sornram/openthaiid/internal/auth"
	"github.com/night-sornram/openthaiid/internal/config"
	"github.com/night-sornram/openthaiid/internal/pcsc"
	"github.com/night-sornram/openthaiid/internal/protocol"
	"github.com/night-sornram/openthaiid/internal/reader"
)

func TestOptionalLoopbackTLSListener(t *testing.T) {
	tb, certFile := startTLSTestBridge(t)
	httpsBase := "https://" + tb.srv.TLSAddr()
	httpBase := tb.httpBase()

	plain := mustJSONGet(t, http.DefaultClient, httpBase+"/healthz")
	if plain["status"] != "ok" {
		t.Fatalf("http healthz %+v", plain)
	}
	if plain["tls"] != true {
		t.Fatalf("http healthz should advertise tls: %+v", plain)
	}
	if plain["wssUrl"] != tb.srv.cfg.WSSURL() {
		t.Fatalf("wssUrl %v", plain["wssUrl"])
	}

	trusted := tlsClient(t, certFile)
	secure := mustJSONGet(t, trusted, httpsBase+"/healthz")
	if secure["status"] != "ok" || secure["tls"] != true {
		t.Fatalf("https healthz %+v", secure)
	}

	untrusted := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    x509.NewCertPool(),
				MinVersion: tls.VersionTLS12,
				ServerName: "127.0.0.1",
			},
		},
	}
	if _, err := untrusted.Get(httpsBase + "/healthz"); err == nil {
		t.Fatal("untrusted client must fail")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, tb.srv.cfg.WSSURL(), &websocket.DialOptions{
		Subprotocols: []string{protocol.Subprotocol},
		HTTPHeader:   http.Header{"Origin": []string{"https://erp.clinic.example"}},
		HTTPClient:   trusted,
	})
	if err != nil {
		t.Fatalf("wss dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	writeReq(t, ctx, conn, 1, "bridge.probe", "", nil)
	msg := readMsg(t, ctx, conn)
	result, _ := msg["result"].(map[string]any)
	if result["tls"] != true {
		t.Fatalf("probe tls %+v", result)
	}
	if result["wssUrl"] != tb.srv.cfg.WSSURL() {
		t.Fatalf("probe wssUrl %+v", result["wssUrl"])
	}

	plainWS := fmt.Sprintf("ws://%s/v1/ws", tb.srv.Addr())
	plainConn, _, err := websocket.Dial(ctx, plainWS, &websocket.DialOptions{
		Subprotocols: []string{protocol.Subprotocol},
		HTTPHeader:   http.Header{"Origin": []string{"http://127.0.0.1:8000"}},
	})
	if err != nil {
		t.Fatalf("plain ws must still work: %v", err)
	}
	plainConn.Close(websocket.StatusNormalClosure, "")

	doc := mustJSONGet(t, http.DefaultClient, httpBase+"/v1/helper/doctor")
	if doc["tls"] != "yes" {
		t.Fatalf("doctor tls %+v", doc)
	}
	rawDoc, _ := json.Marshal(doc)
	if bytes.Contains(rawDoc, []byte(".pem")) || bytes.Contains(rawDoc, []byte("PRIVATE KEY")) || bytes.Contains(rawDoc, []byte("tlsKeyFile")) {
		t.Fatalf("doctor leaked key material: %s", rawDoc)
	}
	text, _ := doc["text"].(string)
	if !strings.Contains(text, "tls=yes") || strings.Contains(text, ".pem") {
		t.Fatalf("doctor text tls: %s", text)
	}
}

func TestTLSStartFailsWithoutCertFiles(t *testing.T) {
	cfg := config.Default()
	cfg.Port = freeLoopbackPort(t)
	cfg.TLSPort = freeLoopbackPort(t)
	cfg.TLSCertFile = filepath.Join(t.TempDir(), "missing.pem")
	cfg.TLSKeyFile = filepath.Join(t.TempDir(), "missing-key.pem")
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	authMgr := auth.NewManager("tls-missing", cfg.TrustDuration, cfg.SessionDuration, cfg.ApprovalTimeout, cfg.MaxPendingApprovals)
	mgr := reader.NewManager(pcsc.NewSynthetic(), pcsc.NewSynthetic())
	s := New(cfg, log, authMgr, mgr)
	if err := s.Start(context.Background()); err == nil {
		t.Fatal("missing cert must fail Start")
	}
}

func startTLSTestBridge(t *testing.T) (*testBridge, string) {
	t.Helper()
	certFile, keyFile := writeLoopbackCert(t, t.TempDir())
	tlsPort := freeLoopbackPort(t)
	tb := startTestBridgeOpts(t, false, nil, func(cfg *config.Config) {
		cfg.TLSCertFile = certFile
		cfg.TLSKeyFile = keyFile
		cfg.TLSPort = tlsPort
	})
	waitHTTP(t, "http://"+tb.srv.Addr()+"/healthz", http.DefaultClient)
	waitHTTP(t, "https://"+tb.srv.TLSAddr()+"/healthz", tlsClient(t, certFile))
	return tb, certFile
}

func writeLoopbackCert(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certFile = filepath.Join(dir, "loopback.pem")
	keyFile = filepath.Join(dir, "loopback-key.pem")
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

func tlsClient(t *testing.T, certFile string) *http.Client {
	t.Helper()
	pemBytes, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pemBytes) {
		t.Fatal("append cert")
	}
	return &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    pool,
				MinVersion: tls.VersionTLS12,
				ServerName: "127.0.0.1",
			},
		},
	}
}

func mustJSONGet(t *testing.T, client *http.Client, url string) map[string]any {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status %d body=%s", url, resp.StatusCode, b)
	}
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
}

func waitHTTP(t *testing.T, url string, client *http.Client) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			last = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("wait %s: %v", url, last)
}

func freeLoopbackPort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil || port < 1 {
		t.Fatalf("port %s", portStr)
	}
	return port
}

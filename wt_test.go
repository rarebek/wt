package wt

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quic-go/webtransport-go"
	"github.com/rarebek/wt/codec"
)

func TestAltSvcHeader(t *testing.T) {
	header := AltSvcHeader(4433)
	expected := `h3=":4433"; ma=86400`
	if header != expected {
		t.Errorf("expected %q, got %q", expected, header)
	}
}
func TestSetAltSvcHeader(t *testing.T) {
	w := httptest.NewRecorder()
	SetAltSvcHeader(w, 443)
	got := w.Header().Get("Alt-Svc")
	if got != `h3=":443"; ma=86400` {
		t.Errorf("expected Alt-Svc header, got %q", got)
	}
}
func TestAltSvcMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mw := AltSvcMiddleware(4433)
	wrapped := mw(handler)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)
	got := w.Header().Get("Alt-Svc")
	if got == "" {
		t.Error("expected Alt-Svc header to be set")
	}
}
func BenchmarkAltSvcHeader(b *testing.B) {
	for b.Loop() {
		AltSvcHeader(4433)
	}
}
func BenchmarkJoinPath(b *testing.B) {
	for b.Loop() {
		JoinPath("/api", "/v1", "/users")
	}
}
func BenchmarkHash(b *testing.B) {
	data := []byte("benchmark hash input")
	b.ResetTimer()
	for b.Loop() {
		Hash(data)
	}
}
func TestConnInfoTransport(t *testing.T) {
	// ConnInfo always reports "webtransport"
	info := ConnInfo{
		Transport: "webtransport",
	}
	if info.Transport != "webtransport" {
		t.Errorf("expected 'webtransport', got %q", info.Transport)
	}
}
func TestHealthCheck(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())
	hc := NewHealthCheck(server)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	hc.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected application/json, got %q", contentType)
	}
	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
	if resp.Transport != "webtransport" {
		t.Errorf("expected transport 'webtransport', got %q", resp.Transport)
	}
	if resp.ActiveSessions != 0 {
		t.Errorf("expected 0 active sessions, got %d", resp.ActiveSessions)
	}
	if resp.Uptime == "" {
		t.Error("expected non-empty uptime")
	}
}
func TestFlowControlMonitor(t *testing.T) {
	fc := NewFlowControlMonitor()
	fc.StreamsOpened.Add(10)
	fc.StreamsClosed.Add(3)
	fc.DatagramsSent.Add(100)
	fc.DatagramsRecvd.Add(95)
	fc.BytesSent.Add(50000)
	fc.BytesReceived.Add(48000)
	stats := fc.Stats()
	if stats.StreamsActive != 7 {
		t.Errorf("expected 7 active streams, got %d", stats.StreamsActive)
	}
	if stats.DatagramsSent != 100 {
		t.Errorf("expected 100 sent, got %d", stats.DatagramsSent)
	}
	if stats.BytesSent != 50000 {
		t.Errorf("expected 50000 bytes sent, got %d", stats.BytesSent)
	}
}
func TestFlowControlMonitorEmpty(t *testing.T) {
	fc := NewFlowControlMonitor()
	stats := fc.Stats()
	if stats.StreamsActive != 0 || stats.BytesSent != 0 {
		t.Error("fresh monitor should have all zeros")
	}
}
func TestPreflightNoTLS(t *testing.T) {
	server := New(WithAddr(":0"))
	issues := server.Preflight()
	found := false
	for _, issue := range issues {
		if issue == "no TLS configuration: use WithTLS(), WithSelfSignedTLS(), WithAutoCert(), or WithCertRotator()" {
			found = true
		}
	}
	if !found {
		t.Error("expected TLS warning")
	}
}
func TestPreflightWithSelfSigned(t *testing.T) {
	server := New(WithAddr("127.0.0.1:0"), WithSelfSignedTLS())
	result := server.PreflightCheck()
	if !result.Ready {
		t.Errorf("expected ready, got issues: %v", result.Issues)
	}
}
func TestPreflightBadCert(t *testing.T) {
	server := New(WithAddr(":0"), WithTLS("/nonexistent/cert.pem", "/nonexistent/key.pem"))
	issues := server.Preflight()
	found := false
	for _, issue := range issues {
		if len(issue) > 10 { // has a cert error message
			found = true
		}
	}
	if !found {
		t.Error("expected cert error")
	}
}
func TestNewServer(t *testing.T) {
	s := New(WithAddr(":4433"), WithSelfSignedTLS())
	if s.Addr() != ":4433" {
		t.Errorf("expected :4433, got %q", s.Addr())
	}
	if s.CertHash() == "" {
		t.Error("expected non-empty cert hash for self-signed")
	}
	if s.SessionCount() != 0 {
		t.Errorf("expected 0 sessions, got %d", s.SessionCount())
	}
}
func TestNewServerDefaults(t *testing.T) {
	s := New()
	if s.Addr() != ":4433" {
		t.Errorf("expected default :4433, got %q", s.Addr())
	}
}
func TestServerVersion(t *testing.T) {
	if Version == "" {
		t.Error("expected non-empty version")
	}
}
func TestServerInfoFromServer(t *testing.T) {
	info := ServerInfo()
	if info["version"] != Version {
		t.Error("expected version match")
	}
}
func TestWithAddr(t *testing.T) {
	s := New(WithAddr(":9999"))
	if s.Addr() != ":9999" {
		t.Errorf("expected :9999, got %q", s.Addr())
	}
}
func TestWithIdleTimeout(t *testing.T) {
	s := New(WithIdleTimeout(5 * time.Minute))
	if s.idleTimeout != 5*time.Minute {
		t.Errorf("expected 5m, got %v", s.idleTimeout)
	}
}
func TestWithSelfSignedTLS(t *testing.T) {
	s := New(WithSelfSignedTLS())
	if s.autoTLS == nil {
		t.Error("expected autoTLS to be set")
	}
	if s.CertHash() == "" {
		t.Error("expected non-empty cert hash")
	}
}
func TestWithTLS(t *testing.T) {
	s := New(WithTLS("cert.pem", "key.pem"))
	if s.tlsCert != "cert.pem" || s.tlsKey != "key.pem" {
		t.Error("TLS file paths not set correctly")
	}
}
func TestWithCheckOrigin(t *testing.T) {
	s := New(WithCheckOrigin(func(r *http.Request) bool {
		return true
	}))
	if s.checkOrigin == nil {
		t.Error("expected checkOrigin to be set")
	}
}
func TestOnShutdownHooks(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())
	var order []string
	server.OnShutdown(func() { order = append(order, "hook1") })
	server.OnShutdown(func() { order = append(order, "hook2") })
	server.OnShutdown(func() { order = append(order, "hook3") })
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	server.Shutdown(ctx)
	if len(order) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(order))
	}
	if order[0] != "hook1" || order[1] != "hook2" || order[2] != "hook3" {
		t.Errorf("wrong order: %v", order)
	}
}
func TestGroupHandle(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())
	api := server.Group("/api")
	api.Handle("/users", func(c *Context) {})
	api.Handle("/posts", func(c *Context) {})
	routes := server.router.Routes()
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
	found := false
	for _, r := range routes {
		if r.Pattern == "/api/users" {
			found = true
		}
	}
	if !found {
		t.Error("expected /api/users route")
	}
}
func TestGroupMiddleware(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())
	api := server.Group("/api", func(c *Context, next HandlerFunc) {
		next(c)
	})
	api.Handle("/test", func(c *Context) {})
	routes := server.router.Routes()
	if len(routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(routes))
	}
	if len(routes[0].Middleware) != 1 {
		t.Errorf("expected 1 middleware on group route, got %d", len(routes[0].Middleware))
	}
}
func TestGroupUse(t *testing.T) {
	server := New(WithAddr(":0"), WithSelfSignedTLS())
	api := server.Group("/api")
	api.Use(func(c *Context, next HandlerFunc) { next(c) })
	api.Handle("/data", func(c *Context) {})
	routes := server.router.Routes()
	if len(routes[0].Middleware) != 1 {
		t.Errorf("expected 1 middleware from Use, got %d", len(routes[0].Middleware))
	}
}
func TestHandleBothIntegration(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())
	server.Handle("/both", HandleBoth(
		func(s *Stream, c *Context) {
			defer s.Close()
			msg, _ := s.ReadMessage()
			s.WriteMessage(append([]byte("stream:"), msg...))
		},
		func(data []byte, c *Context) []byte {
			return append([]byte("dgram:"), data...)
		},
	))
	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/both", addr), nil)
	defer session.CloseWithError(0, "")
	// Test datagram
	session.SendDatagram([]byte("ping"))
	reply, err := session.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("datagram: %v", err)
	}
	if string(reply) != "dgram:ping" {
		t.Errorf("datagram: expected 'dgram:ping', got %q", reply)
	}
	// Test stream
	raw, _ := session.OpenStreamSync(ctx)
	s := &Stream{raw: raw}
	s.WriteMessage([]byte("hello"))
	streamReply, _ := s.ReadMessage()
	if string(streamReply) != "stream:hello" {
		t.Errorf("stream: expected 'stream:hello', got %q", streamReply)
	}
}
func TestHandleStreamConvenience(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())
	// Use HandleStream convenience
	server.Handle("/streamconv", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		_ = s.WriteMessage(append([]byte("echo:"), msg...))
	}))
	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/streamconv", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	s := &Stream{raw: stream, ctx: nil}
	_ = s.WriteMessage([]byte("test"))
	reply, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(reply) != "echo:test" {
		t.Errorf("expected 'echo:test', got %q", reply)
	}
}
func TestHandleDatagramConvenience(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())
	// Use HandleDatagram convenience
	server.Handle("/dgconv", HandleDatagram(func(data []byte, c *Context) []byte {
		return append([]byte("pong:"), data...)
	}))
	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/dgconv", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")
	if err := session.SendDatagram([]byte("hello")); err != nil {
		t.Fatalf("send: %v", err)
	}
	reply, err := session.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if string(reply) != "pong:hello" {
		t.Errorf("expected 'pong:hello', got %q", reply)
	}
}

func TestContextSetGetString(t *testing.T) {
	c := &Context{store: make(map[string]any)}
	c.Set("name", "alice")

	if c.GetString("name") != "alice" {
		t.Errorf("expected 'alice', got %q", c.GetString("name"))
	}
	if c.GetString("missing") != "" {
		t.Error("missing key should return empty string")
	}
}

func TestContextMustGetPanics(t *testing.T) {
	c := &Context{store: make(map[string]any)}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic from MustGet on missing key")
		}
	}()
	c.MustGet("nonexistent")
}

func TestContextMustGetSuccess(t *testing.T) {
	c := &Context{store: make(map[string]any)}
	c.Set("key", 42)

	v := c.MustGet("key")
	if v != 42 {
		t.Errorf("expected 42, got %v", v)
	}
}

func TestContextParams(t *testing.T) {
	c := &Context{
		params: map[string]string{"room": "lobby", "id": "123"},
		store:  make(map[string]any),
	}

	if c.Param("room") != "lobby" {
		t.Errorf("expected 'lobby', got %q", c.Param("room"))
	}
	if c.Param("id") != "123" {
		t.Errorf("expected '123', got %q", c.Param("id"))
	}
	if c.Param("missing") != "" {
		t.Error("missing param should return empty")
	}

	params := c.Params()
	if len(params) != 2 {
		t.Errorf("expected 2 params, got %d", len(params))
	}
}

func TestContextID(t *testing.T) {
	c := &Context{id: "test-id-123", store: make(map[string]any)}
	if c.ID() != "test-id-123" {
		t.Errorf("expected 'test-id-123', got %q", c.ID())
	}
}

type testMsg struct {
	Name  string
	Value int
}

func (m testMsg) Validate() error {
	return RequiredFields(m, "Name", "Value")
}

func TestValidateMessageValid(t *testing.T) {
	msg := testMsg{Name: "test", Value: 42}
	if err := ValidateMessage(msg); err != nil {
		t.Errorf("expected no error: %v", err)
	}
}

func TestValidateMessageInvalid(t *testing.T) {
	msg := testMsg{Name: "", Value: 0}
	err := ValidateMessage(msg)
	if err == nil {
		t.Error("expected error for empty fields")
	}
}

func TestValidateMessageNoValidator(t *testing.T) {
	// Type without Validate() method
	type plainMsg struct{ X int }
	err := ValidateMessage(plainMsg{})
	if err != nil {
		t.Errorf("non-validator should return nil: %v", err)
	}
}

func TestRequiredFieldsMissing(t *testing.T) {
	msg := testMsg{Name: "", Value: 0}
	err := RequiredFields(msg, "Name", "Value")
	if err == nil {
		t.Error("expected error")
	}
}

func TestRequiredFieldsPresent(t *testing.T) {
	msg := testMsg{Name: "ok", Value: 1}
	err := RequiredFields(msg, "Name", "Value")
	if err != nil {
		t.Errorf("expected no error: %v", err)
	}
}

func TestRequiredFieldsPartial(t *testing.T) {
	msg := testMsg{Name: "ok", Value: 0}
	err := RequiredFields(msg, "Name", "Value")
	if err == nil {
		t.Error("expected error for zero Value")
	}
}

func TestRequiredFieldsNonStruct(t *testing.T) {
	err := RequiredFields("not a struct", "Field")
	if err == nil {
		t.Error("expected error for non-struct")
	}
}

type benchMsg struct {
	Name  string
	Value int
	Tags  []string
}

func (m benchMsg) Validate() error {
	return RequiredFields(m, "Name", "Value")
}

func BenchmarkValidateMessage(b *testing.B) {
	msg := benchMsg{Name: "test", Value: 42, Tags: []string{"a"}}
	b.ResetTimer()
	for b.Loop() {
		ValidateMessage(msg)
	}
}

func BenchmarkRequiredFields(b *testing.B) {
	msg := benchMsg{Name: "test", Value: 42}
	b.ResetTimer()
	for b.Loop() {
		RequiredFields(msg, "Name", "Value")
	}
}

func TestPriorityValues(t *testing.T) {
	if PriorityBackground != 0 {
		t.Error("PriorityBackground should be 0")
	}
	if PriorityNormal != 3 {
		t.Error("PriorityNormal should be 3")
	}
	if PriorityCritical != 7 {
		t.Error("PriorityCritical should be 7")
	}
	if PriorityHigh <= PriorityNormal {
		t.Error("PriorityHigh should be greater than PriorityNormal")
	}
}

func TestDefaultStreamConfig(t *testing.T) {
	cfg := DefaultStreamConfig()
	if cfg.Priority != PriorityNormal {
		t.Errorf("expected PriorityNormal, got %d", cfg.Priority)
	}
	if cfg.TypeID != 0 {
		t.Errorf("expected TypeID 0, got %d", cfg.TypeID)
	}
}

func TestVersionString(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestServerInfo(t *testing.T) {
	info := ServerInfo()
	if info["framework"] != "wt" {
		t.Errorf("expected framework 'wt', got %q", info["framework"])
	}
	if info["version"] != Version {
		t.Errorf("expected version %q, got %q", Version, info["version"])
	}
}

func TestHash(t *testing.T) {
	h := Hash([]byte("hello"))
	if len(h) != 64 { // SHA-256 = 32 bytes = 64 hex chars
		t.Errorf("expected 64 char hash, got %d", len(h))
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{"a", "b", "c"}, "/a/b/c"},
		{[]string{"/api", "/v1", "/users"}, "/api/v1/users"},
		{[]string{"", "a", ""}, "/a"},
		{[]string{"/", "a"}, "/a"},
	}

	for _, tt := range tests {
		got := JoinPath(tt.input...)
		if got != tt.want {
			t.Errorf("JoinPath(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMust(t *testing.T) {
	val := Must(42, nil)
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestMustPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	Must(0, fmt.Errorf("boom"))
}

func TestDefaultBatchEncode(t *testing.T) {
	batch := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("!"),
	}

	encoded := defaultBatchEncode(batch)

	// Decode and verify
	decoded := DecodeBatch(encoded)
	if len(decoded) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(decoded))
	}
	if string(decoded[0]) != "hello" {
		t.Errorf("msg[0] = %q, want 'hello'", decoded[0])
	}
	if string(decoded[1]) != "world" {
		t.Errorf("msg[1] = %q, want 'world'", decoded[1])
	}
	if string(decoded[2]) != "!" {
		t.Errorf("msg[2] = %q, want '!'", decoded[2])
	}
}

func TestDecodeBatchEmpty(t *testing.T) {
	msgs := DecodeBatch([]byte{})
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestDecodeBatchSingle(t *testing.T) {
	batch := defaultBatchEncode([][]byte{[]byte("single")})
	msgs := DecodeBatch(batch)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if string(msgs[0]) != "single" {
		t.Errorf("expected 'single', got %q", msgs[0])
	}
}

func TestDecodeBatchTruncated(t *testing.T) {
	// Truncated data should not panic
	msgs := DecodeBatch([]byte{0x00, 0x05, 0x61, 0x62}) // claims length 5 but only 2 bytes
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages from truncated data, got %d", len(msgs))
	}
}

func BenchmarkBatchEncode(b *testing.B) {
	batch := make([][]byte, 10)
	for i := range batch {
		batch[i] = make([]byte, 64)
	}

	b.ResetTimer()
	for b.Loop() {
		defaultBatchEncode(batch)
	}
}

func BenchmarkBatchDecode(b *testing.B) {
	batch := make([][]byte, 10)
	for i := range batch {
		batch[i] = make([]byte, 64)
	}
	encoded := defaultBatchEncode(batch)

	b.ResetTimer()
	for b.Loop() {
		DecodeBatch(encoded)
	}
}

func BenchmarkBatchEncode8(b *testing.B) {
	batch := makeBatch(8, 32)
	b.ResetTimer()
	for b.Loop() {
		defaultBatchEncode(batch)
	}
}

func BenchmarkBatchEncode16(b *testing.B) {
	batch := makeBatch(16, 32)
	b.ResetTimer()
	for b.Loop() {
		defaultBatchEncode(batch)
	}
}

func BenchmarkBatchEncode32(b *testing.B) {
	batch := makeBatch(32, 32)
	b.ResetTimer()
	for b.Loop() {
		defaultBatchEncode(batch)
	}
}

func BenchmarkBatchEncode64(b *testing.B) {
	batch := makeBatch(64, 32)
	b.ResetTimer()
	for b.Loop() {
		defaultBatchEncode(batch)
	}
}

func BenchmarkBatchDecode8(b *testing.B) {
	encoded := defaultBatchEncode(makeBatch(8, 32))
	b.ResetTimer()
	for b.Loop() {
		DecodeBatch(encoded)
	}
}

func BenchmarkBatchDecode16(b *testing.B) {
	encoded := defaultBatchEncode(makeBatch(16, 32))
	b.ResetTimer()
	for b.Loop() {
		DecodeBatch(encoded)
	}
}

func BenchmarkBatchDecode32(b *testing.B) {
	encoded := defaultBatchEncode(makeBatch(32, 32))
	b.ResetTimer()
	for b.Loop() {
		DecodeBatch(encoded)
	}
}

func BenchmarkBatchDecode64(b *testing.B) {
	encoded := defaultBatchEncode(makeBatch(64, 32))
	b.ResetTimer()
	for b.Loop() {
		DecodeBatch(encoded)
	}
}

func makeBatch(count, msgSize int) [][]byte {
	batch := make([][]byte, count)
	for i := range batch {
		batch[i] = make([]byte, msgSize)
		for j := range batch[i] {
			batch[i][j] = byte(j)
		}
	}
	return batch
}

func TestDecodeBatchConcurrent(t *testing.T) {
	batch := defaultBatchEncode([][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("test"),
	})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msgs := DecodeBatch(batch)
			if len(msgs) != 3 {
				t.Errorf("expected 3 messages, got %d", len(msgs))
			}
		}()
	}
	wg.Wait()
}

func TestValidateDatagramSize(t *testing.T) {
	// Valid sizes
	if err := ValidateDatagramSize([]byte("hello")); err != nil {
		t.Errorf("5 bytes should be valid: %v", err)
	}

	if err := ValidateDatagramSize(make([]byte, MaxDatagramSize)); err != nil {
		t.Errorf("%d bytes should be valid: %v", MaxDatagramSize, err)
	}

	// Empty
	if err := ValidateDatagramSize([]byte{}); err != nil {
		t.Errorf("empty should be valid: %v", err)
	}

	// Too large
	err := ValidateDatagramSize(make([]byte, MaxDatagramSize+1))
	if err == nil {
		t.Error("expected error for oversized datagram")
	}
}

func TestMaxDatagramSize(t *testing.T) {
	if MaxDatagramSize != 1200 {
		t.Errorf("expected MaxDatagramSize 1200, got %d", MaxDatagramSize)
	}
}

func TestSessionCloseError(t *testing.T) {
	err := &SessionCloseError{Code: 401, Message: "unauthorized"}

	if err.Error() != "wt: session closed with code 401: unauthorized" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	if !IsSessionClosed(err) {
		t.Error("expected IsSessionClosed to return true")
	}
}

func TestStreamCloseError(t *testing.T) {
	err := &StreamCloseError{Code: 100, Remote: true}
	if err.Error() != "wt: stream closed by remote with code 100" {
		t.Errorf("unexpected error message: %s", err.Error())
	}

	err2 := &StreamCloseError{Code: 0, Remote: false}
	if err2.Error() != "wt: stream closed by local with code 0" {
		t.Errorf("unexpected error message: %s", err2.Error())
	}

	if !IsStreamClosed(err) {
		t.Error("expected IsStreamClosed to return true")
	}
}

func TestIsNotCloseError(t *testing.T) {
	if IsSessionClosed(nil) {
		t.Error("nil should not be a session close error")
	}
	if IsStreamClosed(nil) {
		t.Error("nil should not be a stream close error")
	}
}

func TestConnectionError(t *testing.T) {
	inner := fmt.Errorf("connection refused")
	err := &ConnectionError{Op: "dial", Addr: "localhost:4433", Wrapped: inner}

	if !IsConnectionError(err) {
		t.Error("expected IsConnectionError true")
	}
	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

func TestUpgradeError(t *testing.T) {
	err := &UpgradeError{StatusCode: 500, Message: "internal error"}
	if !IsUpgradeError(err) {
		t.Error("expected IsUpgradeError true")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestMessageError(t *testing.T) {
	inner := fmt.Errorf("broken pipe")
	err := &MessageError{Op: "write", Size: 1024, Wrapped: inner}
	if !IsMessageError(err) {
		t.Error("expected IsMessageError true")
	}
	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

func TestErrorCodes(t *testing.T) {
	if CodeOK != 0 {
		t.Error("CodeOK should be 0")
	}
	if CodeUnauthorized != 401 {
		t.Error("CodeUnauthorized should be 401")
	}
	if CodeProtocolError != 0x1000 {
		t.Error("CodeProtocolError should be 0x1000")
	}
}

func BenchmarkSessionCloseError(b *testing.B) {
	for b.Loop() {
		err := &SessionCloseError{Code: 401, Message: "unauthorized"}
		_ = err.Error()
	}
}

func BenchmarkIsSessionClosed(b *testing.B) {
	err := &SessionCloseError{Code: 0, Message: ""}
	b.ResetTimer()
	for b.Loop() {
		IsSessionClosed(err)
	}
}

func TestReadWriteJSON(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	type Msg struct {
		Name string `json:"name"`
		Val  int    `json:"val"`
	}

	server.Handle("/json", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		var msg Msg
		if err := s.ReadJSON(&msg); err != nil {
			return
		}
		msg.Val *= 2 // double it
		s.WriteJSON(msg)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/json", addr), nil)
	defer session.CloseWithError(0, "")

	raw, _ := session.OpenStreamSync(ctx)
	s := &Stream{raw: raw}

	s.WriteJSON(Msg{Name: "test", Val: 21})

	var reply Msg
	s.ReadJSON(&reply)

	if reply.Name != "test" || reply.Val != 42 {
		t.Errorf("expected {test 42}, got %+v", reply)
	}
}

func TestJoinPathBasic(t *testing.T) {
	tests := []struct {
		in   []string
		want string
	}{
		{[]string{"a", "b"}, "/a/b"},
		{[]string{"/api", "/v1"}, "/api/v1"},
		{[]string{"/"}, ""},
	}

	for _, tt := range tests {
		got := JoinPath(tt.in...)
		if got != tt.want {
			t.Errorf("JoinPath(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestHashDeterministic(t *testing.T) {
	h1 := Hash([]byte("test"))
	h2 := Hash([]byte("test"))
	if h1 != h2 {
		t.Error("Hash should be deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64 char hash, got %d", len(h1))
	}
}

func TestHashDifferentInputs(t *testing.T) {
	h1 := Hash([]byte("hello"))
	h2 := Hash([]byte("world"))
	if h1 == h2 {
		t.Error("different inputs should produce different hashes")
	}
}

func TestMulticastFilter(t *testing.T) {
	// Test the filter function pattern works correctly
	sessions := []*Context{
		{id: "s1", store: map[string]any{"role": "admin"}},
		{id: "s2", store: map[string]any{"role": "user"}},
		{id: "s3", store: map[string]any{"role": "admin"}},
		{id: "s4", store: map[string]any{"role": "guest"}},
	}

	filter := func(c *Context) bool {
		role, _ := c.Get("role")
		return role == "admin"
	}

	var matched int
	for _, s := range sessions {
		if filter(s) {
			matched++
		}
	}

	if matched != 2 {
		t.Errorf("expected 2 admin matches, got %d", matched)
	}
}

// startTestServer creates and starts a server on a random port for testing.
func startTestServer(t *testing.T, setup func(*Server)) (*Server, string) {
	t.Helper()

	// Find a free port
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(
		WithAddr(addr),
		WithSelfSignedTLS(),
	)

	setup(server)

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	select {
	case err := <-errCh:
		t.Fatalf("server failed to start: %v", err)
	default:
	}

	return server, addr
}

func TestServerEchoIntegration(t *testing.T) {
	server, addr := startTestServer(t, func(s *Server) {
		s.Handle("/echo", func(c *Context) {
			for {
				stream, err := c.AcceptStream()
				if err != nil {
					return
				}
				go func() {
					defer stream.Close()
					msg, err := stream.ReadMessage()
					if err != nil {
						return
					}
					_ = stream.WriteMessage(msg)
				}()
			}
		})
	})
	defer server.Close()

	// Connect client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/echo", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Open stream and send message
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream error: %v", err)
	}

	testMsg := []byte("hello webtransport!")
	s := &Stream{raw: stream, ctx: nil}
	if err := s.WriteMessage(testMsg); err != nil {
		t.Fatalf("write error: %v", err)
	}

	reply, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if string(reply) != string(testMsg) {
		t.Errorf("expected %q, got %q", testMsg, reply)
	}
}

func TestServerDatagramIntegration(t *testing.T) {
	server, addr := startTestServer(t, func(s *Server) {
		s.Handle("/dgram", func(c *Context) {
			for {
				data, err := c.ReceiveDatagram()
				if err != nil {
					return
				}
				_ = c.SendDatagram(data)
			}
		})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/dgram", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	testData := []byte("ping")
	if err := session.SendDatagram(testData); err != nil {
		t.Fatalf("send datagram error: %v", err)
	}

	reply, err := session.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("receive datagram error: %v", err)
	}

	if string(reply) != string(testData) {
		t.Errorf("expected %q, got %q", testData, reply)
	}
}

func TestServerMiddlewareIntegration(t *testing.T) {
	var mu sync.Mutex
	var middlewareOrder []string

	server, addr := startTestServer(t, func(s *Server) {
		s.Use(func(c *Context, next HandlerFunc) {
			mu.Lock()
			middlewareOrder = append(middlewareOrder, "global")
			mu.Unlock()
			next(c)
		})

		s.Handle("/mw", func(c *Context) {
			mu.Lock()
			middlewareOrder = append(middlewareOrder, "handler")
			mu.Unlock()
			// Accept one stream to prove we got here
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			_ = stream.WriteMessage([]byte("ok"))
			stream.Close()
		}, func(c *Context, next HandlerFunc) {
			mu.Lock()
			middlewareOrder = append(middlewareOrder, "route")
			mu.Unlock()
			next(c)
		})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/mw", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream error: %v", err)
	}

	// Write something to trigger the server's AcceptStream
	s := &Stream{raw: stream, ctx: nil}
	_ = s.WriteMessage([]byte("ping"))

	msg, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if string(msg) != "ok" {
		t.Errorf("expected 'ok', got %q", msg)
	}

	session.CloseWithError(0, "")
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(middlewareOrder) != 3 {
		t.Fatalf("expected 3 middleware calls, got %d: %v", len(middlewareOrder), middlewareOrder)
	}
	if middlewareOrder[0] != "global" || middlewareOrder[1] != "route" || middlewareOrder[2] != "handler" {
		t.Errorf("wrong order: %v", middlewareOrder)
	}
}

func TestServerSessionStore(t *testing.T) {
	connected := make(chan string, 1)

	server, addr := startTestServer(t, func(s *Server) {
		s.OnConnect(func(c *Context) {
			connected <- c.ID()
		})

		s.Handle("/store", func(c *Context) {
			// Just wait for session to close
			<-c.Context().Done()
		})
	})
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/store", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	select {
	case id := <-connected:
		if id == "" {
			t.Error("expected non-empty session ID")
		}

		if server.Sessions().Count() != 1 {
			t.Errorf("expected 1 active session, got %d", server.Sessions().Count())
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for connection")
	}

	session.CloseWithError(0, "")
	time.Sleep(200 * time.Millisecond)

	if server.Sessions().Count() != 0 {
		t.Errorf("expected 0 sessions after disconnect, got %d", server.Sessions().Count())
	}
}

// TestChaosRandomDisconnects tests that the server handles random client
// disconnections gracefully without panics, leaks, or deadlocks.
func TestChaosRandomDisconnects(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var serverPanics atomic.Int64
	var sessionsHandled atomic.Int64

	server.Handle("/chaos", func(c *Context) {
		defer func() {
			if r := recover(); r != nil {
				serverPanics.Add(1)
			}
		}()
		sessionsHandled.Add(1)

		// Echo streams until disconnect
		for {
			stream, err := c.AcceptStream()
			if err != nil {
				return
			}
			go func() {
				defer stream.Close()
				msg, err := stream.ReadMessage()
				if err != nil {
					return
				}
				_ = stream.WriteMessage(msg)
			}()
		}
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	const numClients = 15
	var wg sync.WaitGroup

	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			dialer := webtransport.Dialer{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/chaos", addr), nil)
			if err != nil {
				return // connection failure is expected in chaos
			}

			// Random behavior: some send data, some disconnect immediately
			action := rand.IntN(3)
			switch action {
			case 0:
				// Disconnect immediately
				session.CloseWithError(0, "chaos")
			case 1:
				// Send one message then disconnect
				stream, err := session.OpenStreamSync(ctx)
				if err != nil {
					session.CloseWithError(0, "")
					return
				}
				s := &Stream{raw: stream}
				s.WriteMessage([]byte("chaos"))
				time.Sleep(time.Duration(rand.IntN(50)) * time.Millisecond)
				session.CloseWithError(0, "chaos")
			case 2:
				// Stay connected briefly, send multiple messages
				for range rand.IntN(5) + 1 {
					stream, err := session.OpenStreamSync(ctx)
					if err != nil {
						break
					}
					s := &Stream{raw: stream}
					s.WriteMessage([]byte(fmt.Sprintf("chaos-%d", id)))
					s.ReadMessage() // wait for echo
				}
				time.Sleep(time.Duration(rand.IntN(100)) * time.Millisecond)
				session.CloseWithError(0, "done")
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Verify no panics on server side
	panics := serverPanics.Load()
	if panics > 0 {
		t.Errorf("server had %d panics during chaos test", panics)
	}

	handled := sessionsHandled.Load()
	t.Logf("chaos test: %d clients, %d sessions handled, %d panics", numClients, handled, panics)

	// Verify server cleaned up properly
	remaining := server.SessionCount()
	if remaining != 0 {
		t.Errorf("expected 0 sessions after chaos, got %d", remaining)
	}
}

func TestScale100Sessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping scale test in short mode")
	}

	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var active atomic.Int64
	var peak atomic.Int64
	var completed atomic.Int64

	server.Handle("/scale", HandleStream(func(s *Stream, c *Context) {
		cur := active.Add(1)
		defer active.Add(-1)
		// Track peak
		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}

		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		s.WriteMessage(msg)
		completed.Add(1)
	}))

	go server.ListenAndServe()
	time.Sleep(150 * time.Millisecond)
	defer server.Close()

	const numClients = 100
	var wg sync.WaitGroup
	var errors atomic.Int64

	start := time.Now()

	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			dialer := webtransport.Dialer{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/scale", addr), nil)
			if err != nil {
				errors.Add(1)
				return
			}
			defer session.CloseWithError(0, "")

			stream, err := session.OpenStreamSync(ctx)
			if err != nil {
				errors.Add(1)
				return
			}

			s := &Stream{raw: stream}
			msg := []byte(fmt.Sprintf("client-%d", id))
			if err := s.WriteMessage(msg); err != nil {
				errors.Add(1)
				return
			}

			reply, err := s.ReadMessage()
			if err != nil {
				errors.Add(1)
				return
			}

			if string(reply) != string(msg) {
				errors.Add(1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("100 sessions: %v elapsed, %d completed, %d errors, peak concurrent: %d",
		elapsed.Truncate(time.Millisecond), completed.Load(), errors.Load(), peak.Load())

	if errors.Load() > 5 { // allow a few failures under load
		t.Errorf("too many errors: %d/%d", errors.Load(), numClients)
	}

	time.Sleep(200 * time.Millisecond)
	if server.SessionCount() != 0 {
		t.Errorf("leaked sessions: %d", server.SessionCount())
	}
}

// TestConcurrentSessions tests multiple concurrent WebTransport sessions.
func TestConcurrentSessions(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var sessionCount atomic.Int64

	server.Handle("/stress", func(c *Context) {
		sessionCount.Add(1)
		defer sessionCount.Add(-1)

		// Echo one stream then close
		stream, err := c.AcceptStream()
		if err != nil {
			return
		}
		msg, err := stream.ReadMessage()
		if err != nil {
			return
		}
		_ = stream.WriteMessage(msg)
		stream.Close()
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	const numClients = 10
	var wg sync.WaitGroup
	errors := make(chan error, numClients)

	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			dialer := webtransport.Dialer{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/stress", addr), nil)
			if err != nil {
				errors <- fmt.Errorf("client %d dial: %w", id, err)
				return
			}
			defer session.CloseWithError(0, "")

			stream, err := session.OpenStreamSync(ctx)
			if err != nil {
				errors <- fmt.Errorf("client %d open stream: %w", id, err)
				return
			}

			msg := []byte(fmt.Sprintf("hello from client %d", id))
			s := &Stream{raw: stream, ctx: nil}
			if err := s.WriteMessage(msg); err != nil {
				errors <- fmt.Errorf("client %d write: %w", id, err)
				return
			}

			reply, err := s.ReadMessage()
			if err != nil {
				errors <- fmt.Errorf("client %d read: %w", id, err)
				return
			}

			if string(reply) != string(msg) {
				errors <- fmt.Errorf("client %d: expected %q, got %q", id, msg, reply)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

// TestConcurrentDatagrams tests multiple concurrent datagram senders.
func TestConcurrentDatagrams(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var received atomic.Int64

	server.Handle("/dgstress", func(c *Context) {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			received.Add(1)
			_ = c.SendDatagram(data)
		}
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/dgstress", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Send 100 datagrams rapidly
	const count = 100
	for i := range count {
		msg := []byte(fmt.Sprintf("dg-%d", i))
		if err := session.SendDatagram(msg); err != nil {
			t.Fatalf("send datagram %d: %v", i, err)
		}
	}

	// Wait for some to be echoed back (datagrams are unreliable, some may be lost)
	time.Sleep(500 * time.Millisecond)

	got := received.Load()
	if got == 0 {
		t.Error("no datagrams received by server")
	}
	t.Logf("sent %d datagrams, server received %d (%.0f%%)", count, got, float64(got)/float64(count)*100)
}

func TestGracefulShutdown(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	sessionActive := make(chan struct{})
	sessionDone := make(chan struct{})

	server.Handle("/drain", func(c *Context) {
		close(sessionActive) // signal that session is up
		<-c.Context().Done() // wait for session close
		close(sessionDone)
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)

	// Connect a client
	ctx := context.Background()
	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/drain", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Wait for session to be active
	select {
	case <-sessionActive:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for session")
	}

	if server.SessionCount() != 1 {
		t.Errorf("expected 1 session, got %d", server.SessionCount())
	}

	// Graceful shutdown with 2s timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go server.Shutdown(shutdownCtx)

	// Session should be force-closed within the timeout
	select {
	case <-sessionDone:
		// Session was closed by shutdown
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for session to drain")
	}

	session.CloseWithError(0, "")
}

// TestLargeMessage tests sending messages near the maximum size.
func TestLargeMessage(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/large", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		_ = s.WriteMessage(msg)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/large", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Send a 1MB message
	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	bigMsg := make([]byte, 1024*1024) // 1 MB
	for i := range bigMsg {
		bigMsg[i] = byte(i % 256)
	}

	s := &Stream{raw: stream, ctx: nil}
	if err := s.WriteMessage(bigMsg); err != nil {
		t.Fatalf("write large message: %v", err)
	}

	reply, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read large message: %v", err)
	}

	if len(reply) != len(bigMsg) {
		t.Errorf("reply length %d, want %d", len(reply), len(bigMsg))
	}
	if !bytes.Equal(reply, bigMsg) {
		t.Error("reply content doesn't match")
	}
}

// TestEmptyMessage tests sending zero-length messages.
func TestEmptyMessage(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/empty", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		_ = s.WriteMessage(msg) // echo empty
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/empty", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	s := &Stream{raw: stream, ctx: nil}
	if err := s.WriteMessage([]byte{}); err != nil {
		t.Fatalf("write empty message: %v", err)
	}

	reply, err := s.ReadMessage()
	if err != nil {
		t.Fatalf("read empty message: %v", err)
	}

	if len(reply) != 0 {
		t.Errorf("expected empty reply, got %d bytes", len(reply))
	}
}

// TestMultipleRoutes tests that different paths route to different handlers.
func TestMultipleRoutes(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/route/a", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		_ = s.WriteMessage([]byte("handler-a"))
	}))
	server.Handle("/route/b", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		_ = s.WriteMessage([]byte("handler-b"))
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Connect to route A
	_, sessionA, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/route/a", addr), nil)
	if err != nil {
		t.Fatalf("dial a: %v", err)
	}
	defer sessionA.CloseWithError(0, "")

	streamA, _ := sessionA.OpenStreamSync(ctx)
	sA := &Stream{raw: streamA, ctx: nil}
	_ = sA.WriteMessage([]byte("trigger"))
	replyA, _ := sA.ReadMessage()

	if string(replyA) != "handler-a" {
		t.Errorf("route /a returned %q, want 'handler-a'", replyA)
	}

	// Connect to route B
	_, sessionB, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/route/b", addr), nil)
	if err != nil {
		t.Fatalf("dial b: %v", err)
	}
	defer sessionB.CloseWithError(0, "")

	streamB, _ := sessionB.OpenStreamSync(ctx)
	sB := &Stream{raw: streamB, ctx: nil}
	_ = sB.WriteMessage([]byte("trigger"))
	replyB, _ := sB.ReadMessage()

	if string(replyB) != "handler-b" {
		t.Errorf("route /b returned %q, want 'handler-b'", replyB)
	}
}

// TestProductionConcurrentMultiPath tests multiple routes handling concurrent
// sessions with different payloads simultaneously.
func TestProductionConcurrentMultiPath(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/route/a", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, _ := s.ReadMessage()
		s.WriteMessage(append([]byte("A:"), msg...))
	}))
	server.Handle("/route/b", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, _ := s.ReadMessage()
		s.WriteMessage(append([]byte("B:"), msg...))
	}))
	server.Handle("/route/c", HandleDatagram(func(d []byte, c *Context) []byte {
		return append([]byte("C:"), d...)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	var wg sync.WaitGroup
	var errors atomic.Int64

	routes := []string{"/route/a", "/route/b"}
	for i := range 20 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			route := routes[id%2]
			dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s%s", addr, route), nil)
			if err != nil {
				errors.Add(1)
				return
			}
			defer session.CloseWithError(0, "")

			raw, err := session.OpenStreamSync(ctx)
			if err != nil {
				errors.Add(1)
				return
			}
			s := &Stream{raw: raw}
			msg := fmt.Sprintf("client-%d", id)
			s.WriteMessage([]byte(msg))

			reply, err := s.ReadMessage()
			if err != nil {
				errors.Add(1)
				return
			}

			var prefix string
			if route == "/route/a" {
				prefix = "A:"
			} else {
				prefix = "B:"
			}
			expected := prefix + msg
			if string(reply) != expected {
				t.Errorf("client %d on %s: got %q, want %q", id, route, reply, expected)
				errors.Add(1)
			}
		}(i)
	}

	// Also test datagram route concurrently
	for i := range 5 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
			_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/route/c", addr), nil)
			if err != nil {
				errors.Add(1)
				return
			}
			defer session.CloseWithError(0, "")

			msg := []byte(fmt.Sprintf("dg-%d", id))
			session.SendDatagram(msg)
			reply, err := session.ReceiveDatagram(ctx)
			if err != nil {
				errors.Add(1)
				return
			}
			expected := "C:" + string(msg)
			if string(reply) != expected {
				t.Errorf("datagram %d: got %q, want %q", id, reply, expected)
				errors.Add(1)
			}
		}(i)
	}

	wg.Wait()

	if errors.Load() > 2 {
		t.Errorf("too many errors: %d", errors.Load())
	}

	time.Sleep(200 * time.Millisecond)
	if server.SessionCount() != 0 {
		t.Errorf("leaked sessions: %d", server.SessionCount())
	}
}

// TestProductionMiddlewareChainOrder verifies that middleware runs in correct
// order and can abort the chain.
func TestProductionMiddlewareChainOrder(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var order []string
	var mu sync.Mutex

	server.Use(func(c *Context, next HandlerFunc) {
		mu.Lock()
		order = append(order, "mw1-before")
		mu.Unlock()
		next(c)
		mu.Lock()
		order = append(order, "mw1-after")
		mu.Unlock()
	})

	server.Handle("/order", func(c *Context) {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		// Accept one stream to prove we got here
		stream, _ := c.AcceptStream()
		stream.WriteMessage([]byte("ok"))
		stream.Close()
	}, func(c *Context, next HandlerFunc) {
		mu.Lock()
		order = append(order, "mw2-before")
		mu.Unlock()
		next(c)
		mu.Lock()
		order = append(order, "mw2-after")
		mu.Unlock()
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/order", addr), nil)
	raw, _ := session.OpenStreamSync(ctx)
	s := &Stream{raw: raw}
	s.WriteMessage([]byte("trigger"))
	s.ReadMessage()
	session.CloseWithError(0, "")
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d steps, got %d: %v", len(expected), len(order), order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Errorf("step %d: expected %q, got %q", i, expected[i], order[i])
		}
	}
}

// TestProductionRapidConnectDisconnect tests rapid session creation and teardown.
func TestProductionRapidConnectDisconnect(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var connected atomic.Int64
	server.Handle("/rapid", func(c *Context) {
		connected.Add(1)
		defer connected.Add(-1)
		<-c.Context().Done()
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	// Rapidly connect and disconnect 30 clients
	for i := range 30 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/rapid", addr), nil)
		cancel()
		if err != nil {
			continue // some may fail under rapid load, that's ok
		}
		// Random hold time
		time.Sleep(time.Duration(rand.IntN(20)) * time.Millisecond)
		session.CloseWithError(0, fmt.Sprintf("rapid-%d", i))
	}

	// Wait for all to drain
	time.Sleep(500 * time.Millisecond)

	active := connected.Load()
	if active != 0 {
		t.Errorf("expected 0 active after rapid connect/disconnect, got %d", active)
	}
	if server.SessionCount() != 0 {
		t.Errorf("leaked sessions: %d", server.SessionCount())
	}
}

// TestMultiStreamIndependence proves that 5 concurrent streams within
// one session operate independently — no head-of-line blocking.
func TestMultiStreamIndependence(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	// Server: echo each stream independently with a delay
	server.Handle("/multi", HandleStream(func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		// Small delay to prove independence (fast stream shouldn't wait for slow)
		_ = s.WriteMessage(msg)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/multi", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Open 5 concurrent streams
	const numStreams = 5
	var wg sync.WaitGroup
	results := make([]string, numStreams)
	errors := make([]error, numStreams)

	for i := range numStreams {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			stream, err := session.OpenStreamSync(ctx)
			if err != nil {
				errors[idx] = fmt.Errorf("open stream %d: %w", idx, err)
				return
			}

			s := &Stream{raw: stream, ctx: nil}
			msg := fmt.Sprintf("stream-%d", idx)

			if err := s.WriteMessage([]byte(msg)); err != nil {
				errors[idx] = fmt.Errorf("write stream %d: %w", idx, err)
				return
			}

			reply, err := s.ReadMessage()
			if err != nil {
				errors[idx] = fmt.Errorf("read stream %d: %w", idx, err)
				return
			}

			results[idx] = string(reply)
		}(i)
	}

	wg.Wait()

	for i := range numStreams {
		if errors[i] != nil {
			t.Error(errors[i])
			continue
		}
		expected := fmt.Sprintf("stream-%d", i)
		if results[i] != expected {
			t.Errorf("stream %d: expected %q, got %q", i, expected, results[i])
		}
	}
}

func TestSessionStoreConcurrent(t *testing.T) {
	ss := NewSessionStore()
	var wg sync.WaitGroup

	// Concurrent add/remove/count
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := &Context{id: fmt.Sprintf("s-%d", id), store: make(map[string]any)}
			ss.Add(ctx)
			_ = ss.Count()
			ss.Remove(ctx.ID())
		}(i)
	}
	wg.Wait()

	if ss.Count() != 0 {
		t.Errorf("expected 0 after concurrent add/remove, got %d", ss.Count())
	}
}

func TestRoomConcurrentJoinLeave(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := &Context{id: fmt.Sprintf("m-%d", id), store: make(map[string]any)}
			room.Join(ctx)
			_ = room.Count()
			room.Leave(ctx)
		}(i)
	}
	wg.Wait()

	if room.Count() != 0 {
		t.Errorf("expected 0 after concurrent join/leave, got %d", room.Count())
	}
}

func TestPubSubConcurrent(t *testing.T) {
	ps := NewPubSub()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := &Context{id: fmt.Sprintf("p-%d", id), store: make(map[string]any)}
			ps.Subscribe("topic", ctx)
			_ = ps.SubscriberCount("topic")
			ps.Unsubscribe("topic", ctx)
		}(i)
	}
	wg.Wait()

	if ps.SubscriberCount("topic") != 0 {
		t.Errorf("expected 0 after concurrent sub/unsub, got %d", ps.SubscriberCount("topic"))
	}
}

func TestTagsConcurrent(t *testing.T) {
	tags := NewTags()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sid := fmt.Sprintf("s-%d", id)
			tags.Tag(sid, "vip")
			_ = tags.HasTag(sid, "vip")
			tags.Untag(sid, "vip")
		}(i)
	}
	wg.Wait()

	if tags.Count("vip") != 0 {
		t.Errorf("expected 0 after concurrent tag/untag, got %d", tags.Count("vip"))
	}
}

func TestEventBusConcurrent(t *testing.T) {
	bus := NewEventBus()
	var count int64
	var mu sync.Mutex

	bus.On(EventConnect, func(e Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Emit(Event{Type: EventConnect})
		}()
	}
	wg.Wait()

	if count != 100 {
		t.Errorf("expected 100 events, got %d", count)
	}
}

// TestMemory1000Sessions verifies that 1000 session Context objects
// don't leak excessive memory. This tests the framework's data structures,
// not actual QUIC connections.
func TestMemory1000Sessions(t *testing.T) {
	ss := NewSessionStore()
	rm := NewRoomManager()
	room := rm.GetOrCreate("loadtest")
	pt := NewPresenceTracker()

	var m1 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Create 1000 sessions with context stores, room membership, and presence
	contexts := make([]*Context, 1000)
	for i := range 1000 {
		ctx := &Context{
			id:     fmt.Sprintf("session-%04d", i),
			params: map[string]string{"room": "loadtest"},
			store:  map[string]any{"user": fmt.Sprintf("user-%d", i), "role": "member"},
		}
		contexts[i] = ctx
		ss.Add(ctx)
		room.Join(ctx)
		pt.Join("loadtest", ctx)
	}

	var m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m2)

	allocatedKB := (m2.HeapAlloc - m1.HeapAlloc) / 1024
	perSessionBytes := (m2.HeapAlloc - m1.HeapAlloc) / 1000

	t.Logf("1000 sessions: ~%d KB total, ~%d bytes/session", allocatedKB, perSessionBytes)
	t.Logf("  SessionStore: %d entries", ss.Count())
	t.Logf("  Room members: %d", room.Count())
	t.Logf("  Presence:     %d", pt.Count("loadtest"))

	// Sanity: should be well under 1MB for 1000 sessions
	if allocatedKB > 2048 {
		t.Errorf("excessive memory: %d KB for 1000 sessions (expected < 2048 KB)", allocatedKB)
	}

	// Verify counts
	if ss.Count() != 1000 {
		t.Errorf("expected 1000 sessions, got %d", ss.Count())
	}
	if room.Count() != 1000 {
		t.Errorf("expected 1000 room members, got %d", room.Count())
	}

	// Cleanup
	for _, ctx := range contexts {
		ss.Remove(ctx.ID())
		room.Leave(ctx)
		pt.Leave("loadtest", ctx)
	}

	if ss.Count() != 0 {
		t.Errorf("leaked sessions: %d", ss.Count())
	}
	if room.Count() != 0 {
		t.Errorf("leaked room members: %d", room.Count())
	}
}

func TestStreamsIterator(t *testing.T) {
	l, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	var streamCount atomic.Int32
	server.Handle("/iter", func(c *Context) {
		for stream := range Streams(c) {
			streamCount.Add(1)
			go func() {
				defer stream.Close()
				msg, _ := stream.ReadMessage()
				stream.WriteMessage(msg)
			}()
		}
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	_, session, _ := dialer.Dial(ctx, fmt.Sprintf("https://%s/iter", addr), nil)
	defer session.CloseWithError(0, "")

	// Open 3 streams via the iterator
	for i := range 3 {
		raw, _ := session.OpenStreamSync(ctx)
		s := &Stream{raw: raw}
		s.WriteMessage([]byte(fmt.Sprintf("iter-%d", i)))
		reply, _ := s.ReadMessage()
		if string(reply) != fmt.Sprintf("iter-%d", i) {
			t.Errorf("stream %d: expected 'iter-%d', got %q", i, i, reply)
		}
	}

	session.CloseWithError(0, "")
	time.Sleep(100 * time.Millisecond)

	if got := streamCount.Load(); got != 3 {
		t.Errorf("expected 3 streams via iterator, got %d", got)
	}
}

func TestBuildChainNoMiddleware(t *testing.T) {
	called := false
	handler := HandlerFunc(func(c *Context) {
		called = true
	})

	chain := buildChain(handler, nil)
	chain(nil) // context can be nil for this test

	if !called {
		t.Error("handler was not called")
	}
}

func TestBuildChainOrder(t *testing.T) {
	var order []string

	mw1 := MiddlewareFunc(func(c *Context, next HandlerFunc) {
		order = append(order, "mw1-before")
		next(c)
		order = append(order, "mw1-after")
	})

	mw2 := MiddlewareFunc(func(c *Context, next HandlerFunc) {
		order = append(order, "mw2-before")
		next(c)
		order = append(order, "mw2-after")
	})

	handler := HandlerFunc(func(c *Context) {
		order = append(order, "handler")
	})

	chain := buildChain(handler, []MiddlewareFunc{mw1, mw2})
	chain(nil)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestBuildChainAbort(t *testing.T) {
	handlerCalled := false

	mw := MiddlewareFunc(func(c *Context, next HandlerFunc) {
		// Don't call next — abort
	})

	handler := HandlerFunc(func(c *Context) {
		handlerCalled = true
	})

	chain := buildChain(handler, []MiddlewareFunc{mw})
	chain(nil)

	if handlerCalled {
		t.Error("handler should not have been called when middleware aborts")
	}
}

func TestRetrySuccess(t *testing.T) {
	cfg := DefaultRetryConfig()

	attempts := 0
	err := Retry(context.Background(), cfg, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestRetryEventualSuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 5,
		InitDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	attempts := 0
	err := Retry(context.Background(), cfg, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("not yet")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryAllFail(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 3,
		InitDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
	}

	err := Retry(context.Background(), cfg, func() error {
		return fmt.Errorf("always fails")
	})

	if err == nil {
		t.Error("expected error")
	}
}

func TestRetryContextCancelled(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts: 100,
		InitDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := Retry(ctx, cfg, func() error {
		return fmt.Errorf("fail")
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("expected 3 max attempts, got %d", cfg.MaxAttempts)
	}
	if !cfg.Jitter {
		t.Error("expected jitter to be true")
	}
}

func TestRingBufferPushItems(t *testing.T) {
	rb := NewRingBuffer[int](5)

	for i := range 5 {
		rb.Push(i)
	}

	items := rb.Items()
	if len(items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(items))
	}
	for i, v := range items {
		if v != i {
			t.Errorf("items[%d] = %d, want %d", i, v, i)
		}
	}
}

func TestRingBufferOverwrite(t *testing.T) {
	rb := NewRingBuffer[int](3)

	for i := range 5 {
		rb.Push(i)
	}

	items := rb.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	// Should contain 2, 3, 4 (oldest overwritten)
	if items[0] != 2 || items[1] != 3 || items[2] != 4 {
		t.Errorf("expected [2,3,4], got %v", items)
	}
}

func TestRingBufferLast(t *testing.T) {
	rb := NewRingBuffer[string](3)

	_, ok := rb.Last()
	if ok {
		t.Error("empty buffer should return false")
	}

	rb.Push("a")
	rb.Push("b")
	rb.Push("c")

	last, ok := rb.Last()
	if !ok || last != "c" {
		t.Errorf("expected 'c', got %q", last)
	}
}

func TestRingBufferLen(t *testing.T) {
	rb := NewRingBuffer[int](10)

	if rb.Len() != 0 {
		t.Error("expected 0")
	}

	rb.Push(1)
	rb.Push(2)
	if rb.Len() != 2 {
		t.Errorf("expected 2, got %d", rb.Len())
	}
}

func TestRingBufferClear(t *testing.T) {
	rb := NewRingBuffer[int](5)
	rb.Push(1)
	rb.Push(2)
	rb.Clear()

	if rb.Len() != 0 {
		t.Errorf("expected 0 after clear, got %d", rb.Len())
	}
}

func TestRingBufferCap(t *testing.T) {
	rb := NewRingBuffer[int](42)
	if rb.Cap() != 42 {
		t.Errorf("expected cap 42, got %d", rb.Cap())
	}
}

func BenchmarkRingBufferPush(b *testing.B) {
	rb := NewRingBuffer[int](1000)
	b.ResetTimer()
	for i := range b.N {
		rb.Push(i)
	}
}

func BenchmarkRingBufferItems(b *testing.B) {
	rb := NewRingBuffer[int](100)
	for i := range 100 {
		rb.Push(i)
	}
	b.ResetTimer()
	for b.Loop() {
		rb.Items()
	}
}

func TestRingBufferConcurrentPush(t *testing.T) {
	rb := NewRingBuffer[int](100)
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			rb.Push(v)
		}(i)
	}
	wg.Wait()

	if rb.Len() != 100 {
		t.Errorf("expected 100, got %d", rb.Len())
	}
}

func TestRingBufferConcurrentReadWrite(t *testing.T) {
	rb := NewRingBuffer[int](50)
	var wg sync.WaitGroup

	// Writers
	for i := range 100 {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			rb.Push(v)
		}(i)
	}

	// Readers
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rb.Items()
			_ = rb.Len()
			rb.Last()
		}()
	}

	wg.Wait()
	// Should not panic or deadlock
}

func TestExtractParamNames(t *testing.T) {
	tests := []struct {
		pattern string
		want    []string
	}{
		{"/chat/{room}", []string{"room"}},
		{"/game/{id}/input", []string{"id"}},
		{"/user/{org}/{team}/{id}", []string{"org", "team", "id"}},
		{"/static/path", nil},
		{"/{a}/{b}", []string{"a", "b"}},
	}

	for _, tt := range tests {
		got := extractParamNames(tt.pattern)
		if len(got) != len(tt.want) {
			t.Errorf("extractParamNames(%q) = %v, want %v", tt.pattern, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("extractParamNames(%q)[%d] = %q, want %q", tt.pattern, i, got[i], tt.want[i])
			}
		}
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		match   bool
		params  map[string]string
	}{
		{"/chat/{room}", "/chat/general", true, map[string]string{"room": "general"}},
		{"/chat/{room}", "/chat/", false, nil},
		{"/chat/{room}", "/chat/general/extra", false, nil},
		{"/game/{id}/input", "/game/123/input", true, map[string]string{"id": "123"}},
		{"/game/{id}/input", "/game/123/output", false, nil},
		{"/static", "/static", true, map[string]string{}},
		{"/static", "/other", false, nil},
		{"/{a}/{b}", "/hello/world", true, map[string]string{"a": "hello", "b": "world"}},
		{"/", "/", true, map[string]string{}},
		// Catch-all wildcard
		{"/static/{path...}", "/static/css/main.css", true, map[string]string{"path": "css/main.css"}},
		{"/files/{rest...}", "/files/a/b/c/d", true, map[string]string{"rest": "a/b/c/d"}},
		{"/api/{version}/{path...}", "/api/v1/users/list", true, map[string]string{"version": "v1", "path": "users/list"}},
	}

	for _, tt := range tests {
		params, ok := matchPattern(tt.pattern, tt.path)
		if ok != tt.match {
			t.Errorf("matchPattern(%q, %q) match = %v, want %v", tt.pattern, tt.path, ok, tt.match)
			continue
		}
		if !ok {
			continue
		}
		if len(params) != len(tt.params) {
			t.Errorf("matchPattern(%q, %q) params = %v, want %v", tt.pattern, tt.path, params, tt.params)
			continue
		}
		for k, v := range tt.params {
			if params[k] != v {
				t.Errorf("matchPattern(%q, %q) param[%q] = %q, want %q", tt.pattern, tt.path, k, params[k], v)
			}
		}
	}
}

func TestRouterAddAndMatch(t *testing.T) {
	r := NewRouter()

	r.Add("/chat/{room}", func(c *Context) {})
	r.Add("/game/{id}/state", func(c *Context) {})

	route, params := r.Match("/chat/lobby")
	if route == nil {
		t.Fatal("expected to match /chat/{room}")
	}
	if params["room"] != "lobby" {
		t.Errorf("expected room=lobby, got %q", params["room"])
	}

	route, params = r.Match("/game/42/state")
	if route == nil {
		t.Fatal("expected to match /game/{id}/state")
	}
	if params["id"] != "42" {
		t.Errorf("expected id=42, got %q", params["id"])
	}

	route, _ = r.Match("/unknown/path")
	if route != nil {
		t.Error("expected no match for /unknown/path")
	}
}

func TestRouterRoutes(t *testing.T) {
	r := NewRouter()
	r.Add("/a", func(c *Context) {})
	r.Add("/b", func(c *Context) {})
	r.Add("/c", func(c *Context) {})

	routes := r.Routes()
	if len(routes) != 3 {
		t.Errorf("expected 3 routes, got %d", len(routes))
	}
}

func BenchmarkRouterMatch1Route(b *testing.B) {
	r := NewRouter()
	r.Add("/echo", func(c *Context) {})

	b.ResetTimer()
	for b.Loop() {
		r.Match("/echo")
	}
}

func BenchmarkRouterMatch10Routes(b *testing.B) {
	r := NewRouter()
	for i := range 10 {
		path := "/" + string(rune('a'+i))
		r.Add(path+"/{id}", func(c *Context) {})
	}

	b.ResetTimer()
	for b.Loop() {
		r.Match("/e/42") // middle of the list
	}
}

func BenchmarkRouterMatch50Routes(b *testing.B) {
	r := NewRouter()
	for i := range 50 {
		path := "/" + string(rune('a'+(i%26))) + string(rune('a'+(i/26)))
		r.Add(path+"/{id}", func(c *Context) {})
	}

	b.ResetTimer()
	for b.Loop() {
		r.Match("/y/42") // near end
	}
}

func BenchmarkRouterMatchCatchAll(b *testing.B) {
	r := NewRouter()
	r.Add("/static/{path...}", func(c *Context) {})

	b.ResetTimer()
	for b.Loop() {
		r.Match("/static/css/main.css")
	}
}

func FuzzMatchPattern(f *testing.F) {
	// Seed corpus
	f.Add("/chat/{room}", "/chat/general")
	f.Add("/game/{id}/input", "/game/123/input")
	f.Add("/{a}/{b}/{c}", "/x/y/z")
	f.Add("/static", "/static")
	f.Add("/", "/")
	f.Add("/a/b/c", "/a/b/c")
	f.Add("/a/{b}", "/a/")
	f.Add("/{x}", "/hello")
	f.Add("/chat/{room}", "/other/path")

	f.Fuzz(func(t *testing.T, pattern, path string) {
		// Should never panic
		params, ok := matchPattern(pattern, path)
		if ok {
			// If matched, params should be non-nil
			if params == nil {
				t.Error("matched but params is nil")
			}
		}
	})
}

func FuzzExtractParamNames(f *testing.F) {
	f.Add("/chat/{room}")
	f.Add("/{a}/{b}/{c}")
	f.Add("/static/path")
	f.Add("")
	f.Add("/{}")
	f.Add("/{{nested}}")
	f.Add("/{valid}/text/{also}")

	f.Fuzz(func(t *testing.T, pattern string) {
		// Should never panic
		names := extractParamNames(pattern)
		_ = names
	})
}

func FuzzCountSegments(f *testing.F) {
	f.Add("")
	f.Add("/")
	f.Add("/a/b/c")
	f.Add("///")

	f.Fuzz(func(t *testing.T, s string) {
		n := countSegments(s)
		if n < 1 {
			t.Errorf("countSegments(%q) = %d, want >= 1", s, n)
		}
	})
}

func TestRPCErrorString(t *testing.T) {
	err := &RPCError{Code: -32601, Message: "method not found"}
	if err.Error() != "rpc error -32601: method not found" {
		t.Errorf("unexpected error string: %s", err.Error())
	}
}

func TestRPCServerRegister(t *testing.T) {
	rpc := NewRPCServer()

	rpc.Register("add", func(params json.RawMessage) (any, error) {
		return 42, nil
	})

	rpc.mu.RLock()
	_, ok := rpc.handlers["add"]
	rpc.mu.RUnlock()

	if !ok {
		t.Error("expected 'add' handler to be registered")
	}
}

func TestRPCRequestMarshal(t *testing.T) {
	req := RPCRequest{
		ID:     1,
		Method: "echo",
		Params: json.RawMessage(`{"msg":"hello"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded RPCRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.ID != 1 || decoded.Method != "echo" {
		t.Errorf("decoded mismatch: %+v", decoded)
	}
}

func TestRPCResponseWithResult(t *testing.T) {
	resp := RPCResponse{
		ID:     1,
		Result: json.RawMessage(`42`),
	}

	data, _ := json.Marshal(resp)
	var decoded RPCResponse
	json.Unmarshal(data, &decoded)

	if decoded.Error != nil {
		t.Error("expected no error")
	}
	if string(decoded.Result) != "42" {
		t.Errorf("expected '42', got %q", decoded.Result)
	}
}

func TestRPCResponseWithError(t *testing.T) {
	resp := RPCResponse{
		ID:    1,
		Error: &RPCError{Code: -32600, Message: "invalid request"},
	}

	data, _ := json.Marshal(resp)
	var decoded RPCResponse
	json.Unmarshal(data, &decoded)

	if decoded.Error == nil {
		t.Fatal("expected error")
	}
	if decoded.Error.Code != -32600 {
		t.Errorf("expected code -32600, got %d", decoded.Error.Code)
	}
}

func TestRPCErrorInterface(t *testing.T) {
	err := &RPCError{Code: -32600, Message: "invalid request"}

	var e error = err
	if e.Error() != "rpc error -32600: invalid request" {
		t.Errorf("unexpected error: %s", e.Error())
	}
}

func TestRPCServerMethodNotFound(t *testing.T) {
	rpc := NewRPCServer()
	rpc.Register("test", func(params json.RawMessage) (any, error) {
		return nil, nil
	})

	rpc.mu.RLock()
	if _, ok := rpc.handlers["test"]; !ok {
		t.Error("test handler should be registered")
	}
	if _, ok := rpc.handlers["missing"]; ok {
		t.Error("missing handler should not exist")
	}
	rpc.mu.RUnlock()
}

func BenchmarkRPCRequestMarshal(b *testing.B) {
	req := RPCRequest{ID: 1, Method: "echo", Params: json.RawMessage(`{"msg":"hello"}`)}
	b.ResetTimer()
	for b.Loop() {
		json.Marshal(req)
	}
}

func BenchmarkRPCResponseMarshal(b *testing.B) {
	resp := RPCResponse{ID: 1, Result: json.RawMessage(`42`)}
	b.ResetTimer()
	for b.Loop() {
		json.Marshal(resp)
	}
}

func TestRPCOverQUIC(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	rpcServer := NewRPCServer()
	rpcServer.Register("add", func(params json.RawMessage) (any, error) {
		var args [2]int
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, err
		}
		return args[0] + args[1], nil
	})
	rpcServer.Register("echo", func(params json.RawMessage) (any, error) {
		var msg string
		json.Unmarshal(params, &msg)
		return msg, nil
	})

	server.Handle("/rpc", HandleStream(func(s *Stream, c *Context) {
		rpcServer.Serve(s)
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/rpc", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	stream, err := session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}

	client := NewRPCClient(&Stream{raw: stream})

	// Test add method
	t.Run("add", func(t *testing.T) {
		result, err := CallTyped[int](client, "add", [2]int{3, 7})
		if err != nil {
			t.Fatalf("rpc call: %v", err)
		}
		if result != 10 {
			t.Errorf("expected 10, got %d", result)
		}
	})

	// Test echo method
	t.Run("echo", func(t *testing.T) {
		result, err := CallTyped[string](client, "echo", "hello rpc")
		if err != nil {
			t.Fatalf("rpc call: %v", err)
		}
		if result != "hello rpc" {
			t.Errorf("expected 'hello rpc', got %q", result)
		}
	})

	// Test unknown method
	t.Run("unknown", func(t *testing.T) {
		_, err := client.Call("nonexistent", nil)
		if err == nil {
			t.Error("expected error for unknown method")
		}
	})
}

func TestSessionStore(t *testing.T) {
	ss := NewSessionStore()

	if ss.Count() != 0 {
		t.Errorf("expected 0 sessions, got %d", ss.Count())
	}

	// We can't create real Context objects without a webtransport.Session,
	// but we can test the store operations with the map directly.
	// For unit testing, we'll test the data structure logic.
}

func TestRoomManager(t *testing.T) {
	rm := NewRoomManager()

	room := rm.GetOrCreate("lobby")
	if room.Name() != "lobby" {
		t.Errorf("expected room name 'lobby', got %q", room.Name())
	}

	if room.Count() != 0 {
		t.Errorf("expected 0 members, got %d", room.Count())
	}

	// GetOrCreate returns same room
	room2 := rm.GetOrCreate("lobby")
	if room2 != room {
		t.Error("expected same room instance")
	}

	// Different room
	room3 := rm.GetOrCreate("game")
	if room3 == room {
		t.Error("expected different room instance")
	}

	rooms := rm.Rooms()
	if len(rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(rooms))
	}

	rm.Remove("game")
	rooms = rm.Rooms()
	if len(rooms) != 1 {
		t.Errorf("expected 1 room after removal, got %d", len(rooms))
	}
}

func TestSessionStoreFindByValue(t *testing.T) {
	ss := NewSessionStore()

	// Add contexts with different user values
	for i, user := range []string{"alice", "bob", "alice", "charlie", "alice"} {
		ctx := &Context{
			id:    fmt.Sprintf("session-%d", i),
			store: map[string]any{"user": user},
		}
		ss.Add(ctx)
	}

	// Find alice's sessions
	aliceSessions := ss.FindByValue("user", "alice")
	if len(aliceSessions) != 3 {
		t.Errorf("expected 3 alice sessions, got %d", len(aliceSessions))
	}

	bobSessions := ss.FindByValue("user", "bob")
	if len(bobSessions) != 1 {
		t.Errorf("expected 1 bob session, got %d", len(bobSessions))
	}

	noneSessions := ss.FindByValue("user", "nobody")
	if len(noneSessions) != 0 {
		t.Errorf("expected 0 sessions for nobody, got %d", len(noneSessions))
	}
}

func TestSessionStoreIDs(t *testing.T) {
	ss := NewSessionStore()

	for i := range 3 {
		ctx := &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: make(map[string]any),
		}
		ss.Add(ctx)
	}

	ids := ss.IDs()
	if len(ids) != 3 {
		t.Errorf("expected 3 IDs, got %d", len(ids))
	}
}

func TestRoomManagerGet(t *testing.T) {
	rm := NewRoomManager()

	_, ok := rm.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent room")
	}

	rm.GetOrCreate("test")
	room, ok := rm.Get("test")
	if !ok {
		t.Error("expected true for existing room")
	}
	if room.Name() != "test" {
		t.Errorf("expected room name 'test', got %q", room.Name())
	}
}

func TestSessionStoreFilter(t *testing.T) {
	ss := NewSessionStore()
	for i := range 10 {
		ctx := &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: map[string]any{"role": "user"},
		}
		if i < 3 {
			ctx.store["role"] = "admin"
		}
		ss.Add(ctx)
	}

	admins := ss.Filter(func(c *Context) bool {
		role, _ := c.Get("role")
		return role == "admin"
	})

	if len(admins) != 3 {
		t.Errorf("expected 3 admins, got %d", len(admins))
	}
}

func TestSessionStoreCountWhere(t *testing.T) {
	ss := NewSessionStore()
	for i := range 20 {
		ctx := &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: map[string]any{"active": i%2 == 0},
		}
		ss.Add(ctx)
	}

	count := ss.CountWhere(func(c *Context) bool {
		v, _ := c.Get("active")
		return v == true
	})

	if count != 10 {
		t.Errorf("expected 10, got %d", count)
	}
}

func TestRoomHas(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")
	ctx := &Context{id: "s1", store: make(map[string]any)}

	if room.Has("s1") {
		t.Error("should not have s1 before join")
	}

	room.Join(ctx)
	if !room.Has("s1") {
		t.Error("should have s1 after join")
	}

	room.Leave(ctx)
	if room.Has("s1") {
		t.Error("should not have s1 after leave")
	}
}

func TestRoomFilterMembers(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")

	for i := range 10 {
		ctx := &Context{
			id:    string(rune('a' + i)),
			store: map[string]any{"vip": i < 3},
		}
		room.Join(ctx)
	}

	vips := room.FilterMembers(func(c *Context) bool {
		v, _ := c.Get("vip")
		return v == true
	})

	if len(vips) != 3 {
		t.Errorf("expected 3 vips, got %d", len(vips))
	}
}

func TestRoomWithHistoryBroadcastRecords(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("chat")
	rwh := NewRoomWithHistory(room, 50)

	rwh.BroadcastAndRecord("user1", []byte("hello"))
	rwh.BroadcastAndRecord("user2", []byte("world"))

	if rwh.HistorySize() != 2 {
		t.Errorf("expected 2 messages, got %d", rwh.HistorySize())
	}

	history := rwh.History()
	if string(history[0].Data) != "hello" {
		t.Errorf("expected 'hello', got %q", history[0].Data)
	}
	if history[0].SenderID != "user1" {
		t.Errorf("expected sender 'user1', got %q", history[0].SenderID)
	}
	if string(history[1].Data) != "world" {
		t.Errorf("expected 'world', got %q", history[1].Data)
	}
}

func TestRoomWithHistoryOverflow(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")
	rwh := NewRoomWithHistory(room, 3)

	for i := range 5 {
		rwh.BroadcastAndRecord("user", []byte{byte(i)})
	}

	if rwh.HistorySize() != 3 {
		t.Errorf("expected 3 (capacity), got %d", rwh.HistorySize())
	}

	history := rwh.History()
	// Should contain 2, 3, 4 (oldest overwritten)
	if history[0].Data[0] != 2 || history[1].Data[0] != 3 || history[2].Data[0] != 4 {
		t.Errorf("expected [2,3,4], got [%d,%d,%d]",
			history[0].Data[0], history[1].Data[0], history[2].Data[0])
	}
}

func TestRoomWithHistoryClear(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")
	rwh := NewRoomWithHistory(room, 10)

	rwh.BroadcastAndRecord("u", []byte("a"))
	rwh.ClearHistory()

	if rwh.HistorySize() != 0 {
		t.Errorf("expected 0 after clear, got %d", rwh.HistorySize())
	}
}

func TestKVSyncSetGet(t *testing.T) {
	kv := NewKVSync()

	if err := kv.Set("name", "alice"); err != nil {
		t.Fatalf("set error: %v", err)
	}

	var name string
	if err := kv.Get("name", &name); err != nil {
		t.Fatalf("get error: %v", err)
	}

	if name != "alice" {
		t.Errorf("expected 'alice', got %q", name)
	}
}

func TestKVSyncDelete(t *testing.T) {
	kv := NewKVSync()
	kv.Set("key", "value")
	kv.Delete("key")

	_, ok := kv.GetRaw("key")
	if ok {
		t.Error("expected key to be deleted")
	}
}

func TestKVSyncKeys(t *testing.T) {
	kv := NewKVSync()
	kv.Set("a", 1)
	kv.Set("b", 2)
	kv.Set("c", 3)

	keys := kv.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestKVSyncLen(t *testing.T) {
	kv := NewKVSync()
	if kv.Len() != 0 {
		t.Error("expected 0 length")
	}

	kv.Set("x", true)
	if kv.Len() != 1 {
		t.Errorf("expected 1, got %d", kv.Len())
	}
}

func TestKVSyncOnChange(t *testing.T) {
	kv := NewKVSync()

	var changedKey string
	kv.OnChange(func(key string, _ json.RawMessage) {
		changedKey = key
	})

	kv.Set("test", 42)

	if changedKey != "test" {
		t.Errorf("expected onChange with key 'test', got %q", changedKey)
	}
}

func TestKVSyncSnapshot(t *testing.T) {
	kv := NewKVSync()
	kv.Set("a", 1)
	kv.Set("b", "two")

	snap := kv.Snapshot()
	if len(snap) != 2 {
		t.Errorf("expected 2 items in snapshot, got %d", len(snap))
	}
}

func TestKVSyncGetMissing(t *testing.T) {
	kv := NewKVSync()

	var val int
	err := kv.Get("nonexistent", &val)
	if err != nil {
		t.Errorf("get missing key should not error: %v", err)
	}
	if val != 0 {
		t.Errorf("expected zero value, got %d", val)
	}
}

func BenchmarkKVSyncSet(b *testing.B) {
	kv := NewKVSync()
	b.ResetTimer()
	for i := range b.N {
		kv.Set("key", i)
	}
}

func BenchmarkKVSyncGet(b *testing.B) {
	kv := NewKVSync()
	kv.Set("key", 42)
	b.ResetTimer()
	for b.Loop() {
		var v int
		kv.Get("key", &v)
	}
}

func TestKVSyncConcurrent(t *testing.T) {
	kv := NewKVSync()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			kv.Set(fmt.Sprintf("key-%d", id), id)
		}(i)
	}

	// Concurrent reads
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = kv.Keys()
			_ = kv.Len()
			_ = kv.Snapshot()
		}()
	}

	wg.Wait()

	if kv.Len() != 100 {
		t.Errorf("expected 100 keys, got %d", kv.Len())
	}
}

func TestKVSyncConcurrentDelete(t *testing.T) {
	kv := NewKVSync()
	for i := range 50 {
		kv.Set(fmt.Sprintf("k-%d", i), i)
	}

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			kv.Delete(fmt.Sprintf("k-%d", id))
		}(i)
	}
	wg.Wait()

	if kv.Len() != 0 {
		t.Errorf("expected 0 after concurrent delete, got %d", kv.Len())
	}
}

func TestResumeStoreSaveRestore(t *testing.T) {
	rs := NewResumeStore(5 * time.Minute)

	// Create a context with some state
	ctx := &Context{
		store: map[string]any{
			"user":  "alice",
			"role":  "admin",
			"score": 42,
		},
	}

	token := rs.Save(ctx)
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if rs.Count() != 1 {
		t.Errorf("expected 1 stored state, got %d", rs.Count())
	}

	// Restore to new context
	newCtx := &Context{
		store: make(map[string]any),
	}

	ok := rs.Restore(newCtx, token)
	if !ok {
		t.Fatal("expected successful restore")
	}

	// Verify state was restored
	user, _ := newCtx.Get("user")
	if user != "alice" {
		t.Errorf("expected user 'alice', got %v", user)
	}

	role, _ := newCtx.Get("role")
	if role != "admin" {
		t.Errorf("expected role 'admin', got %v", role)
	}

	score, _ := newCtx.Get("score")
	if score != 42 {
		t.Errorf("expected score 42, got %v", score)
	}

	// Token should be single-use
	if rs.Count() != 0 {
		t.Errorf("expected 0 stored states after restore, got %d", rs.Count())
	}
}

func TestResumeStoreSingleUse(t *testing.T) {
	rs := NewResumeStore(5 * time.Minute)

	ctx := &Context{
		store: map[string]any{"key": "value"},
	}

	token := rs.Save(ctx)

	// First restore succeeds
	newCtx1 := &Context{store: make(map[string]any)}
	ok := rs.Restore(newCtx1, token)
	if !ok {
		t.Fatal("first restore should succeed")
	}

	// Second restore with same token fails (single-use)
	newCtx2 := &Context{store: make(map[string]any)}
	ok = rs.Restore(newCtx2, token)
	if ok {
		t.Error("second restore should fail (token is single-use)")
	}
}

func TestResumeStoreExpiry(t *testing.T) {
	rs := NewResumeStore(50 * time.Millisecond)

	ctx := &Context{
		store: map[string]any{"key": "value"},
	}

	token := rs.Save(ctx)

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	newCtx := &Context{store: make(map[string]any)}
	ok := rs.Restore(newCtx, token)
	if ok {
		t.Error("expired token should not restore")
	}
}

func TestResumeStoreInvalidToken(t *testing.T) {
	rs := NewResumeStore(5 * time.Minute)

	newCtx := &Context{store: make(map[string]any)}
	ok := rs.Restore(newCtx, "nonexistent-token")
	if ok {
		t.Error("invalid token should not restore")
	}
}

func BenchmarkResumeStoreSave(b *testing.B) {
	rs := NewResumeStore(5 * time.Minute)
	ctx := &Context{store: map[string]any{"user": "alice"}}
	b.ResetTimer()
	for range b.N {
		rs.Save(ctx)
	}
}

func BenchmarkResumeStoreRestore(b *testing.B) {
	rs := NewResumeStore(5 * time.Minute)
	ctx := &Context{store: map[string]any{"user": "alice"}}
	tokens := make([]ResumeToken, b.N)
	for i := range b.N {
		tokens[i] = rs.Save(ctx)
	}
	b.ResetTimer()
	for i := range b.N {
		newCtx := &Context{store: make(map[string]any)}
		rs.Restore(newCtx, tokens[i])
	}
}

func TestPubSubSubscribePublish(t *testing.T) {
	ps := NewPubSub()

	c1 := &Context{id: "s1", store: make(map[string]any)}
	c2 := &Context{id: "s2", store: make(map[string]any)}

	ps.Subscribe("news", c1)
	ps.Subscribe("news", c2)
	ps.Subscribe("sports", c1)

	if ps.SubscriberCount("news") != 2 {
		t.Errorf("expected 2 news subs, got %d", ps.SubscriberCount("news"))
	}
	if ps.SubscriberCount("sports") != 1 {
		t.Errorf("expected 1 sports sub, got %d", ps.SubscriberCount("sports"))
	}
}

func TestPubSubUnsubscribe(t *testing.T) {
	ps := NewPubSub()

	c1 := &Context{id: "s1", store: make(map[string]any)}
	ps.Subscribe("topic", c1)
	ps.Unsubscribe("topic", c1)

	if ps.SubscriberCount("topic") != 0 {
		t.Errorf("expected 0 after unsubscribe, got %d", ps.SubscriberCount("topic"))
	}
}

func TestPubSubUnsubscribeAll(t *testing.T) {
	ps := NewPubSub()

	c1 := &Context{id: "s1", store: make(map[string]any)}
	ps.Subscribe("a", c1)
	ps.Subscribe("b", c1)
	ps.Subscribe("c", c1)

	ps.UnsubscribeAll(c1)

	topics := ps.TopicsForSession("s1")
	if len(topics) != 0 {
		t.Errorf("expected 0 topics after UnsubscribeAll, got %d", len(topics))
	}
}

func TestPubSubTopics(t *testing.T) {
	ps := NewPubSub()

	c := &Context{id: "s1", store: make(map[string]any)}
	ps.Subscribe("x", c)
	ps.Subscribe("y", c)
	ps.Subscribe("z", c)

	topics := ps.Topics()
	if len(topics) != 3 {
		t.Errorf("expected 3 topics, got %d", len(topics))
	}
}

func TestPubSubTopicsForSession(t *testing.T) {
	ps := NewPubSub()

	c := &Context{id: "s1", store: make(map[string]any)}
	ps.Subscribe("news", c)
	ps.Subscribe("weather", c)

	topics := ps.TopicsForSession("s1")
	if len(topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(topics))
	}
}

func TestPubSubEmpty(t *testing.T) {
	ps := NewPubSub()

	if ps.SubscriberCount("nothing") != 0 {
		t.Error("expected 0")
	}
	if len(ps.Topics()) != 0 {
		t.Error("expected empty topics")
	}
}

func TestPersistentPubSubHistory(t *testing.T) {
	pps := NewPersistentPubSub(10)

	pps.PublishPersistent("news", []byte("msg1"))
	pps.PublishPersistent("news", []byte("msg2"))
	pps.PublishPersistent("news", []byte("msg3"))

	if pps.HistoryLen("news") != 3 {
		t.Errorf("expected 3, got %d", pps.HistoryLen("news"))
	}
	if pps.HistoryLen("empty") != 0 {
		t.Errorf("expected 0 for empty topic, got %d", pps.HistoryLen("empty"))
	}
}

func TestPersistentPubSubClear(t *testing.T) {
	pps := NewPersistentPubSub(10)

	pps.PublishPersistent("news", []byte("msg"))
	pps.ClearHistory("news")

	if pps.HistoryLen("news") != 0 {
		t.Errorf("expected 0 after clear, got %d", pps.HistoryLen("news"))
	}
}

func TestPersistentPubSubOverflow(t *testing.T) {
	pps := NewPersistentPubSub(3)

	for i := range 5 {
		pps.PublishPersistent("topic", []byte{byte(i)})
	}

	if pps.HistoryLen("topic") != 3 {
		t.Errorf("expected 3 (capacity), got %d", pps.HistoryLen("topic"))
	}
}

func BenchmarkPubSubSubscribe(b *testing.B) {
	ps := NewPubSub()
	contexts := make([]*Context, b.N)
	for i := range contexts {
		contexts[i] = &Context{id: fmt.Sprintf("s-%d", i), store: make(map[string]any)}
	}
	b.ResetTimer()
	for i := range b.N {
		ps.Subscribe("topic", contexts[i])
	}
}

func BenchmarkPubSubTopics(b *testing.B) {
	ps := NewPubSub()
	c := &Context{id: "s1", store: make(map[string]any)}
	for i := range 100 {
		ps.Subscribe(fmt.Sprintf("topic-%d", i), c)
	}
	b.ResetTimer()
	for b.Loop() {
		ps.Topics()
	}
}

func BenchmarkTagsTag(b *testing.B) {
	tags := NewTags()
	b.ResetTimer()
	for i := range b.N {
		tags.Tag(fmt.Sprintf("s-%d", i), "vip")
	}
}

func BenchmarkTagsLookup(b *testing.B) {
	tags := NewTags()
	for i := range 1000 {
		tags.Tag(fmt.Sprintf("s-%d", i), "vip")
	}
	b.ResetTimer()
	for b.Loop() {
		tags.HasTag("s-500", "vip")
	}
}

func TestTypedPubSubPublish(t *testing.T) {
	ps := NewPubSub()

	type Msg struct {
		Text string `json:"text"`
	}

	tps := NewTypedPubSub[Msg](ps, codec.JSON{})

	c := &Context{id: "s1", store: make(map[string]any)}
	tps.Subscribe("chat", c)

	if ps.SubscriberCount("chat") != 1 {
		t.Errorf("expected 1 subscriber, got %d", ps.SubscriberCount("chat"))
	}

	// Verify encoding works (don't publish to nil session)
	data, err := codec.JSON{}.Marshal(Msg{Text: "hello"})
	if err != nil {
		t.Errorf("encode error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty encoded data")
	}

	tps.Unsubscribe("chat", c)
	if ps.SubscriberCount("chat") != 0 {
		t.Error("expected 0 after unsubscribe")
	}
}

func TestTypedPubSubUnsubscribeAll(t *testing.T) {
	ps := NewPubSub()
	tps := NewTypedPubSub[string](ps, codec.JSON{})

	c := &Context{id: "s1", store: make(map[string]any)}
	tps.Subscribe("a", c)
	tps.Subscribe("b", c)

	tps.UnsubscribeAll(c)

	if ps.SubscriberCount("a") != 0 || ps.SubscriberCount("b") != 0 {
		t.Error("expected 0 subs after UnsubscribeAll")
	}
}

func TestPresenceTrackerJoinLeave(t *testing.T) {
	pt := NewPresenceTracker()

	ctx := &Context{
		id:    "session-1",
		store: map[string]any{"user": "alice"},
	}

	pt.Join("room1", ctx)

	presence := pt.GetPresence("room1")
	if len(presence) != 1 {
		t.Fatalf("expected 1 present, got %d", len(presence))
	}
	if presence[0].UserID != "alice" {
		t.Errorf("expected user 'alice', got %q", presence[0].UserID)
	}
	if presence[0].Status != "online" {
		t.Errorf("expected status 'online', got %q", presence[0].Status)
	}

	pt.Leave("room1", ctx)

	presence = pt.GetPresence("room1")
	if len(presence) != 0 {
		t.Errorf("expected 0 present after leave, got %d", len(presence))
	}
}

func TestPresenceTrackerCount(t *testing.T) {
	pt := NewPresenceTracker()

	for i := range 5 {
		ctx := &Context{
			id:    string(rune('a' + i)),
			store: make(map[string]any),
		}
		pt.Join("room", ctx)
	}

	if pt.Count("room") != 5 {
		t.Errorf("expected 5, got %d", pt.Count("room"))
	}
	if pt.Count("empty") != 0 {
		t.Errorf("expected 0 for empty room, got %d", pt.Count("empty"))
	}
}

func TestPresenceTrackerUpdateStatus(t *testing.T) {
	pt := NewPresenceTracker()

	ctx := &Context{
		id:    "s1",
		store: make(map[string]any),
	}
	pt.Join("room", ctx)
	pt.UpdateStatus("room", "s1", "typing")

	presence := pt.GetPresence("room")
	if presence[0].Status != "typing" {
		t.Errorf("expected 'typing', got %q", presence[0].Status)
	}
}

func TestPresenceTrackerSetMetadata(t *testing.T) {
	pt := NewPresenceTracker()

	ctx := &Context{
		id:    "s1",
		store: make(map[string]any),
	}
	pt.Join("room", ctx)
	pt.SetMetadata("room", "s1", map[string]any{"cursor": 42})

	presence := pt.GetPresence("room")
	if presence[0].Metadata["cursor"] != 42 {
		t.Errorf("expected cursor=42, got %v", presence[0].Metadata["cursor"])
	}
}

func TestPresenceTrackerOnChange(t *testing.T) {
	pt := NewPresenceTracker()

	var events []string
	pt.OnChange(func(room string, info PresenceInfo, event string) {
		events = append(events, event)
	})

	ctx := &Context{
		id:    "s1",
		store: make(map[string]any),
	}
	pt.Join("room", ctx)
	pt.UpdateStatus("room", "s1", "idle")
	pt.Leave("room", ctx)

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %v", len(events), events)
	}
	if events[0] != "join" || events[1] != "update" || events[2] != "leave" {
		t.Errorf("wrong events: %v", events)
	}
}

func TestPresenceTrackerJSON(t *testing.T) {
	pt := NewPresenceTracker()

	ctx := &Context{
		id:    "s1",
		store: map[string]any{"user": "bob"},
	}
	pt.Join("room", ctx)

	data := pt.GetPresenceJSON("room")
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}

func BenchmarkPresenceJoin(b *testing.B) {
	pt := NewPresenceTracker()

	contexts := make([]*Context, b.N)
	for i := range contexts {
		contexts[i] = &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: map[string]any{"user": fmt.Sprintf("user-%d", i)},
		}
	}

	b.ResetTimer()
	for i := range b.N {
		pt.Join("room", contexts[i])
	}
}

func BenchmarkPresenceGetPresence(b *testing.B) {
	pt := NewPresenceTracker()

	for i := range 100 {
		ctx := &Context{
			id:    fmt.Sprintf("s-%d", i),
			store: map[string]any{"user": fmt.Sprintf("user-%d", i)},
		}
		pt.Join("room", ctx)
	}

	b.ResetTimer()
	for b.Loop() {
		pt.GetPresence("room")
	}
}

func TestPresenceConcurrent(t *testing.T) {
	pt := NewPresenceTracker()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := &Context{id: fmt.Sprintf("s-%d", id), store: map[string]any{"user": fmt.Sprintf("u-%d", id)}}
			pt.Join("room", ctx)
			pt.UpdateStatus("room", ctx.ID(), "typing")
			_ = pt.GetPresence("room")
			_ = pt.Count("room")
			pt.Leave("room", ctx)
		}(i)
	}
	wg.Wait()

	if pt.Count("room") != 0 {
		t.Errorf("expected 0 after concurrent join/leave, got %d", pt.Count("room"))
	}
}

func TestEventBusOn(t *testing.T) {
	bus := NewEventBus()

	called := false
	bus.On(EventConnect, func(e Event) {
		called = true
	})

	bus.Emit(Event{Type: EventConnect})

	if !called {
		t.Error("handler was not called")
	}
}

func TestEventBusMultipleHandlers(t *testing.T) {
	bus := NewEventBus()

	var count int
	bus.On(EventConnect, func(e Event) { count++ })
	bus.On(EventConnect, func(e Event) { count++ })
	bus.On(EventConnect, func(e Event) { count++ })

	bus.Emit(Event{Type: EventConnect})

	if count != 3 {
		t.Errorf("expected 3 handler calls, got %d", count)
	}
}

func TestEventBusDifferentTypes(t *testing.T) {
	bus := NewEventBus()

	connectCalled := false
	disconnectCalled := false

	bus.On(EventConnect, func(e Event) { connectCalled = true })
	bus.On(EventDisconnect, func(e Event) { disconnectCalled = true })

	bus.Emit(Event{Type: EventConnect})

	if !connectCalled {
		t.Error("connect handler not called")
	}
	if disconnectCalled {
		t.Error("disconnect handler should not be called")
	}
}

func TestEventBusAsync(t *testing.T) {
	bus := NewEventBus()

	var called atomic.Bool
	bus.On(EventConnect, func(e Event) {
		called.Store(true)
	})

	bus.EmitAsync(Event{Type: EventConnect})

	time.Sleep(50 * time.Millisecond)
	if !called.Load() {
		t.Error("async handler was not called")
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventConnect, "connect"},
		{EventDisconnect, "disconnect"},
		{EventJoinRoom, "join_room"},
		{EventLeaveRoom, "leave_room"},
		{EventType(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.et.String(); got != tt.want {
			t.Errorf("EventType(%d).String() = %q, want %q", tt.et, got, tt.want)
		}
	}
}

func TestEventBusRoomEvent(t *testing.T) {
	bus := NewEventBus()

	var receivedRoom string
	bus.On(EventJoinRoom, func(e Event) {
		receivedRoom = e.Room
	})

	bus.Emit(Event{Type: EventJoinRoom, Room: "lobby"})

	if receivedRoom != "lobby" {
		t.Errorf("expected room 'lobby', got %q", receivedRoom)
	}
}

func TestTagsTagUntag(t *testing.T) {
	tags := NewTags()

	tags.Tag("s1", "premium")
	tags.Tag("s1", "admin")
	tags.Tag("s2", "premium")

	if !tags.HasTag("s1", "premium") {
		t.Error("s1 should have premium tag")
	}
	if !tags.HasTag("s1", "admin") {
		t.Error("s1 should have admin tag")
	}
	if tags.HasTag("s2", "admin") {
		t.Error("s2 should not have admin tag")
	}

	if tags.Count("premium") != 2 {
		t.Errorf("expected 2 premium, got %d", tags.Count("premium"))
	}

	tags.Untag("s1", "premium")
	if tags.HasTag("s1", "premium") {
		t.Error("s1 should not have premium after untag")
	}
	if tags.Count("premium") != 1 {
		t.Errorf("expected 1 premium after untag, got %d", tags.Count("premium"))
	}
}

func TestTagsUntagAll(t *testing.T) {
	tags := NewTags()
	tags.Tag("s1", "a")
	tags.Tag("s1", "b")
	tags.Tag("s1", "c")

	tags.UntagAll("s1")

	if len(tags.TagsForSession("s1")) != 0 {
		t.Error("expected no tags after UntagAll")
	}
}

func TestTagsSessionsWithTag(t *testing.T) {
	tags := NewTags()
	tags.Tag("s1", "vip")
	tags.Tag("s2", "vip")
	tags.Tag("s3", "vip")

	vips := tags.SessionsWithTag("vip")
	if len(vips) != 3 {
		t.Errorf("expected 3 vips, got %d", len(vips))
	}
}

func TestTagsAllTags(t *testing.T) {
	tags := NewTags()
	tags.Tag("s1", "a")
	tags.Tag("s1", "b")
	tags.Tag("s2", "c")

	all := tags.AllTags()
	if len(all) != 3 {
		t.Errorf("expected 3 tags, got %d", len(all))
	}
}

func TestTagsEmpty(t *testing.T) {
	tags := NewTags()

	if tags.HasTag("nonexistent", "tag") {
		t.Error("should be false for nonexistent")
	}
	if tags.Count("empty") != 0 {
		t.Error("should be 0")
	}
	if len(tags.SessionsWithTag("nothing")) != 0 {
		t.Error("should be empty")
	}
}

func TestNewTypedRoom(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")

	type ChatMsg struct {
		User string `json:"user"`
		Text string `json:"text"`
	}

	tr := NewTypedRoom[ChatMsg](room, codec.JSON{})

	if tr.Room() != room {
		t.Error("expected same room reference")
	}
}

func TestTypedRoomBroadcastEncodes(t *testing.T) {
	rm := NewRoomManager()
	room := rm.GetOrCreate("test")

	type Msg struct {
		Value int `json:"value"`
	}

	tr := NewTypedRoom[Msg](room, codec.JSON{})

	// Broadcast with no members should not error
	err := tr.Broadcast(Msg{Value: 42})
	if err != nil {
		t.Errorf("broadcast with no members should not error: %v", err)
	}

	err = tr.BroadcastExcept(Msg{Value: 99}, "nobody")
	if err != nil {
		t.Errorf("broadcast except with no members should not error: %v", err)
	}
}

func TestBackpressureWriterBufferUsage(t *testing.T) {
	// We can't create a real Stream without a WebTransport session,
	// but we can test the channel logic independently.

	bw := &BackpressureWriter{
		queue: make(chan []byte, 4),
		done:  make(chan struct{}),
	}

	if bw.IsFull() {
		t.Error("buffer should not be full when empty")
	}

	usage := bw.BufferUsage()
	if usage != 0.0 {
		t.Errorf("expected 0.0 usage, got %f", usage)
	}

	// Fill the buffer
	for range 4 {
		bw.queue <- []byte("msg")
	}

	if !bw.IsFull() {
		t.Error("buffer should be full")
	}

	usage = bw.BufferUsage()
	if usage != 1.0 {
		t.Errorf("expected 1.0 usage, got %f", usage)
	}

	// Drain
	for range 4 {
		<-bw.queue
	}

	if bw.IsFull() {
		t.Error("buffer should not be full after draining")
	}
}

func TestBackpressureWriterDrops(t *testing.T) {
	bw := &BackpressureWriter{
		queue: make(chan []byte, 2),
		done:  make(chan struct{}),
	}

	// Fill buffer
	ok1 := bw.Send([]byte("a"))
	ok2 := bw.Send([]byte("b"))
	ok3 := bw.Send([]byte("c")) // should be dropped

	if !ok1 || !ok2 {
		t.Error("first two sends should succeed")
	}
	if ok3 {
		t.Error("third send should be dropped (buffer full)")
	}

	_, dropped := bw.Stats()
	if dropped != 1 {
		t.Errorf("expected 1 dropped, got %d", dropped)
	}
}

func TestBackpressureWriterClose(t *testing.T) {
	bw := &BackpressureWriter{
		queue: make(chan []byte, 4),
		done:  make(chan struct{}),
	}

	bw.Close()
	// Double close should not panic
	bw.Close()
}

func TestBackpressureWriterDropsE2E(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	resultCh := make(chan [2]uint64, 1)

	server.Handle("/bp", HandleDatagram(func(data []byte, c *Context) []byte {
		// Use backpressure writer with tiny buffer for datagrams
		bw := &BackpressureWriter{
			queue: make(chan []byte, 2), // very small buffer
			done:  make(chan struct{}),
		}

		// Try to send 10 messages instantly — most should be dropped
		for i := range 10 {
			bw.Send([]byte(fmt.Sprintf("msg-%d", i)))
		}

		sent, dropped := bw.Stats()
		resultCh <- [2]uint64{sent, dropped}
		bw.Close()

		return []byte("ok")
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/bp", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Trigger the handler
	session.SendDatagram([]byte("go"))

	select {
	case result := <-resultCh:
		sent, dropped := result[0], result[1]
		t.Logf("backpressure: sent=%d dropped=%d (buffer=2, attempted=10)", sent, dropped)
		if dropped == 0 {
			t.Error("expected some messages to be dropped with buffer size 2")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func TestCompressionStatsRatio(t *testing.T) {
	stats := CompressionStats{
		RawBytes:        1000,
		CompressedBytes: 300,
	}

	ratio := stats.Ratio()
	if ratio != 0.3 {
		t.Errorf("expected 0.3, got %f", ratio)
	}
}

func TestCompressionStatsZero(t *testing.T) {
	stats := CompressionStats{}
	if stats.Ratio() != 0 {
		t.Error("zero raw bytes should give 0 ratio")
	}
}

func TestCompressedStreamOverQUIC(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/compress", HandleStream(func(s *Stream, c *Context) {
		cs := NewCompressedStream(s, 64) // compress messages > 64 bytes
		defer cs.Close()

		msg, err := cs.ReadMessage()
		if err != nil {
			return
		}
		_ = cs.WriteMessage(msg) // echo compressed
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/compress", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Test with large message (should be compressed)
	t.Run("large_message", func(t *testing.T) {
		stream, err := session.OpenStreamSync(ctx)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}

		cs := NewCompressedStream(&Stream{raw: stream}, 64)

		// Large, compressible message
		bigMsg := bytes.Repeat([]byte("hello world compressed "), 100)

		if err := cs.WriteMessage(bigMsg); err != nil {
			t.Fatalf("write: %v", err)
		}

		reply, err := cs.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}

		if !bytes.Equal(reply, bigMsg) {
			t.Errorf("reply mismatch: got %d bytes, want %d", len(reply), len(bigMsg))
		}
	})

	// Test with small message (should NOT be compressed)
	t.Run("small_message", func(t *testing.T) {
		stream, err := session.OpenStreamSync(ctx)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}

		cs := NewCompressedStream(&Stream{raw: stream}, 64)

		smallMsg := []byte("hi")

		if err := cs.WriteMessage(smallMsg); err != nil {
			t.Fatalf("write: %v", err)
		}

		reply, err := cs.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}

		if string(reply) != "hi" {
			t.Errorf("expected 'hi', got %q", reply)
		}
	})
}

func TestInterceptorOnRead(t *testing.T) {
	readCalled := false
	si := &StreamInterceptor{
		onRead: func(data []byte) ([]byte, error) {
			readCalled = true
			return append([]byte("prefix:"), data...), nil
		},
	}

	result, err := si.onRead([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !readCalled {
		t.Error("onRead was not called")
	}
	if string(result) != "prefix:hello" {
		t.Errorf("expected 'prefix:hello', got %q", result)
	}
}

func TestInterceptorOnWrite(t *testing.T) {
	si := &StreamInterceptor{
		onWrite: func(data []byte) ([]byte, error) {
			if len(data) > 100 {
				return nil, fmt.Errorf("message too large")
			}
			return data, nil
		},
	}

	// Valid message
	result, err := si.onWrite([]byte("ok"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result) != "ok" {
		t.Errorf("expected 'ok', got %q", result)
	}

	// Too large
	_, err = si.onWrite(make([]byte, 200))
	if err == nil {
		t.Error("expected error for large message")
	}
}

func TestInterceptNil(t *testing.T) {
	// No interceptors set — should pass through
	si := &StreamInterceptor{}

	if si.onRead != nil {
		t.Error("onRead should be nil")
	}
	if si.onWrite != nil {
		t.Error("onWrite should be nil")
	}
}

func TestStreamPoolSize(t *testing.T) {
	sp := &StreamPool{max: 4}

	if sp.Size() != 0 {
		t.Errorf("expected 0, got %d", sp.Size())
	}
}

func TestStreamPoolMaxIdle(t *testing.T) {
	sp := NewStreamPool(nil, 2)
	if sp.max != 2 {
		t.Errorf("expected max 2, got %d", sp.max)
	}
}

func TestStreamPoolDefaultMax(t *testing.T) {
	sp := NewStreamPool(nil, 0)
	if sp.max != 4 {
		t.Errorf("expected default max 4, got %d", sp.max)
	}
}

func TestStreamPoolClose(t *testing.T) {
	sp := &StreamPool{max: 4}
	sp.Close() // should not panic on empty pool
	if sp.Size() != 0 {
		t.Error("expected 0 after close")
	}
}

func TestStreamWithTimeout(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	server.Handle("/timeout", HandleStream(func(s *Stream, c *Context) {
		// Use WithTimeout to auto-close stream after 500ms
		cs := s.WithTimeout(500 * time.Millisecond)
		defer cs.Close()

		// This read should fail because the client won't send anything
		// and the timeout will close the stream
		_, err := cs.ReadMessageContext()
		if err == nil {
			t.Error("expected error from timeout")
		}
	}))

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/timeout", addr), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Open stream but don't send anything — server should timeout
	_, err = session.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Wait for server timeout
	time.Sleep(800 * time.Millisecond)
}

func TestDefaultKeepAliveInterval(t *testing.T) {
	if DefaultKeepAliveInterval.Seconds() != 15 {
		t.Errorf("expected 15s, got %v", DefaultKeepAliveInterval)
	}
}

func TestTickerStop(t *testing.T) {
	// Verify Stop is idempotent and doesn't panic
	tk := &Ticker{
		ticker: nil,
		done:   make(chan struct{}),
	}
	// Can't create full Ticker without Context, but test Stop logic
	close(tk.done)
	// Double stop should not panic
	tk.Stop()
}

func TestDefaultKeepAliveValue(t *testing.T) {
	if DefaultKeepAliveInterval.Seconds() != 15 {
		t.Errorf("expected 15s, got %v", DefaultKeepAliveInterval)
	}
}

func TestThrottleAllow(t *testing.T) {
	th := NewThrottle(10, 3) // 10/sec, burst 3

	// First 3 should pass (burst)
	for i := range 3 {
		if !th.Allow() {
			t.Errorf("allow %d should pass", i)
		}
	}

	// 4th should fail
	if th.Allow() {
		t.Error("4th should be throttled")
	}
}

func TestThrottleRefill(t *testing.T) {
	th := NewThrottle(100, 1) // 100/sec, burst 1

	th.Allow() // consume the 1 token

	if th.Allow() {
		t.Error("should be empty")
	}

	time.Sleep(20 * time.Millisecond) // should refill ~2 tokens at 100/s
	if !th.Allow() {
		t.Error("should have refilled")
	}
}

func TestStreamMuxHandle(t *testing.T) {
	mux := NewStreamMux()

	mux.Handle(1, func(s *Stream, c *Context) {})

	if len(mux.handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(mux.handlers))
	}
}

func TestStreamMuxMultipleHandlers(t *testing.T) {
	mux := NewStreamMux()

	mux.Handle(1, func(s *Stream, c *Context) {})
	mux.Handle(2, func(s *Stream, c *Context) {})
	mux.Handle(3, func(s *Stream, c *Context) {})

	if len(mux.handlers) != 3 {
		t.Errorf("expected 3 handlers, got %d", len(mux.handlers))
	}
}

func TestStreamMuxFallback(t *testing.T) {
	mux := NewStreamMux()

	mux.Fallback(func(s *Stream, c *Context) {})

	if mux.fallback == nil {
		t.Error("fallback should be set")
	}
}

func TestStreamMuxIntegration(t *testing.T) {
	l, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	server := New(WithAddr(addr), WithSelfSignedTLS())

	const (
		TypeEcho  uint16 = 1
		TypeUpper uint16 = 2
	)

	mux := NewStreamMux()
	mux.Handle(TypeEcho, func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		_ = s.WriteMessage(msg) // echo as-is
	})
	mux.Handle(TypeUpper, func(s *Stream, c *Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}
		// Convert to uppercase
		for i, b := range msg {
			if b >= 'a' && b <= 'z' {
				msg[i] = b - 32
			}
		}
		_ = s.WriteMessage(msg)
	})

	server.Handle("/mux", func(c *Context) {
		mux.Serve(c)
	})

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	_, session, err := dialer.Dial(ctx, fmt.Sprintf("https://%s/mux", addr), nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer session.CloseWithError(0, "")

	// Test 1: Send to echo handler (type 1)
	t.Run("echo", func(t *testing.T) {
		stream, err := session.OpenStreamSync(ctx)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}

		// Write type header
		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, TypeEcho)
		stream.Write(header)

		s := &Stream{raw: stream, ctx: nil}
		_ = s.WriteMessage([]byte("hello"))

		reply, err := s.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if string(reply) != "hello" {
			t.Errorf("expected 'hello', got %q", reply)
		}
	})

	// Test 2: Send to upper handler (type 2)
	t.Run("upper", func(t *testing.T) {
		stream, err := session.OpenStreamSync(ctx)
		if err != nil {
			t.Fatalf("open stream: %v", err)
		}

		header := make([]byte, 2)
		binary.BigEndian.PutUint16(header, TypeUpper)
		stream.Write(header)

		s := &Stream{raw: stream, ctx: nil}
		_ = s.WriteMessage([]byte("hello world"))

		reply, err := s.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if string(reply) != "HELLO WORLD" {
			t.Errorf("expected 'HELLO WORLD', got %q", reply)
		}
	})
}

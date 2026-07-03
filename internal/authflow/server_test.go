package authflow

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// freePort returns an OS-assigned free port on 127.0.0.1.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("freePort: close: %v", err)
	}
	return port
}

// occupy binds port so it looks busy to Listen; the caller's cleanup closes it.
func occupy(t *testing.T, port int) {
	t.Helper()
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("occupy port %d: %v", port, err)
	}
	t.Cleanup(func() { _ = ln.Close() })
}

// mustListen binds ports or fails the test, closing the listener on cleanup.
func mustListen(t *testing.T, ports []int) *Listener {
	t.Helper()
	l, err := Listen(ports)
	if err != nil {
		t.Fatalf("Listen() error: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}

type callbackOutcome struct {
	result *CallbackResult
	err    error
}

// waitAsync runs WaitForCallback in a goroutine so callers send it concurrently.
func waitAsync(ctx context.Context, l *Listener, state string) <-chan callbackOutcome {
	ch := make(chan callbackOutcome, 1)
	go func() {
		r, err := l.WaitForCallback(ctx, state)
		ch <- callbackOutcome{r, err}
	}()
	return ch
}

// TestListen covers loopback-only binding, port fallback, and all-busy.
func TestListen(t *testing.T) {
	t.Run("binds loopback only, never wildcard", func(t *testing.T) {
		port := freePort(t)
		l := mustListen(t, []int{port})
		tcpAddr, ok := l.Addr().(*net.TCPAddr)
		if !ok || !tcpAddr.IP.IsLoopback() || tcpAddr.IP.IsUnspecified() || tcpAddr.IP.String() != "127.0.0.1" {
			t.Errorf("bound addr = %v, want exactly 127.0.0.1 (never 0.0.0.0)", l.Addr())
		}
		if l.Port() != port {
			t.Errorf("Port() = %d, want %d", l.Port(), port)
		}
	})

	t.Run("falls back to next port when first is busy", func(t *testing.T) {
		busy := freePort(t)
		occupy(t, busy)
		free := freePort(t)
		l := mustListen(t, []int{busy, free})
		if l.Port() != free {
			t.Errorf("Port() = %d, want fallback port %d", l.Port(), free)
		}
	})

	t.Run("all ports busy errors before any browser call", func(t *testing.T) {
		p1, p2 := freePort(t), freePort(t)
		occupy(t, p1)
		occupy(t, p2)
		l, err := Listen([]int{p1, p2})
		if l != nil {
			t.Errorf("Listen() = %v, want nil on all-busy", l)
		}
		busyErr, ok := err.(*PortsBusyError)
		if !ok {
			t.Fatalf("err = %v (%T), want *PortsBusyError", err, err)
		}
		if len(busyErr.Ports) != 2 || busyErr.Ports[0] != p1 || busyErr.Ports[1] != p2 {
			t.Errorf("PortsBusyError.Ports = %v, want [%d %d]", busyErr.Ports, p1, p2)
		}
	})
}

// TestListener_WaitForCallback_Scenarios covers success, state mismatch
// (CSRF), IdP denial, and a malformed callback.
func TestListener_WaitForCallback_Scenarios(t *testing.T) {
	tests := []struct {
		name           string
		expectedState  string
		query          string
		wantCode       string
		wantErrType    string // "" | "state" | "idp" | "callback"
		wantIdPCode    string
		assertShutdown bool // second GET after success must be refused (single-use)
	}{
		{
			name:           "valid code and matching state",
			expectedState:  "expected-state",
			query:          "code=abc123&state=expected-state",
			wantCode:       "abc123",
			assertShutdown: true,
		},
		{
			name:          "state mismatch is rejected",
			expectedState: "expected-state",
			query:         "code=super-secret-code&state=wrong-state",
			wantErrType:   "state",
		},
		{
			name:          "IdP denial is detected before any exchange",
			expectedState: "expected-state",
			query:         "error=access_denied&error_description=User+denied+access&state=expected-state",
			wantErrType:   "idp",
			wantIdPCode:   "access_denied",
		},
		{
			name:          "missing code with matching state is a callback error",
			expectedState: "expected-state",
			query:         "state=expected-state",
			wantErrType:   "callback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := freePort(t)
			l := mustListen(t, []int{port})
			resultCh := waitAsync(context.Background(), l, tt.expectedState)
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?%s", port, tt.query))
			if err != nil {
				t.Fatalf("GET /callback: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			out := <-resultCh
			bodyStr := string(body)
			if !strings.Contains(bodyStr, "close this tab") {
				t.Errorf("response body does not contain close-tab guidance: %q", bodyStr)
			}
			if strings.Contains(bodyStr, "super-secret-code") || strings.Contains(bodyStr, "wrong-state") {
				t.Error("response body leaked a secret query value (code/state)")
			}
			switch tt.wantErrType {
			case "":
				if out.err != nil {
					t.Fatalf("unexpected error: %v", out.err)
				}
				if out.result == nil || out.result.Code != tt.wantCode {
					t.Errorf("result = %+v, want Code %q", out.result, tt.wantCode)
				}
			case "state":
				if _, ok := out.err.(*StateMismatchError); !ok {
					t.Fatalf("err = %v (%T), want *StateMismatchError", out.err, out.err)
				}
				if out.result != nil {
					t.Errorf("result = %+v, want nil on state mismatch", out.result)
				}
			case "idp":
				idpErr, ok := out.err.(*IdPError)
				if !ok {
					t.Fatalf("err = %v (%T), want *IdPError", out.err, out.err)
				}
				if idpErr.Code != tt.wantIdPCode {
					t.Errorf("IdPError.Code = %q, want %q", idpErr.Code, tt.wantIdPCode)
				}
				if out.result != nil {
					t.Errorf("result = %+v, want nil on IdP denial", out.result)
				}
			case "callback":
				if _, ok := out.err.(*CallbackError); !ok {
					t.Fatalf("err = %v (%T), want *CallbackError", out.err, out.err)
				}
			}
			// assertShutdown: post-return, the port must refuse a second GET.
			if tt.assertShutdown {
				if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=second&state=%s", port, tt.expectedState)); err == nil {
					t.Error("second request after shutdown succeeded, want connection error")
				}
			}
		})
	}
}

// TestListener_WaitForCallback_Timeout proves shutdown when ctx is done.
func TestListener_WaitForCallback_Timeout(t *testing.T) {
	port := freePort(t)
	l := mustListen(t, []int{port})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := l.WaitForCallback(ctx, "expected-state")
	if _, ok := err.(*TimeoutError); !ok {
		t.Fatalf("err = %v (%T), want *TimeoutError", err, err)
	}

	if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback", port)); err == nil {
		t.Error("request after timeout shutdown succeeded, want connection error")
	}
}

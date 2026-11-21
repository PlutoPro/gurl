package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// ---- WebSocket ----

func newWSEchoServer(t *testing.T) string {
	t.Helper()
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/forbidden" {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if h := r.Header.Get("X-Token"); h != "" {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("token="+h))
		}
		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			_ = conn.WriteMessage(mt, append([]byte("echo:"), msg...))
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func TestWebSocketSendAndEcho(t *testing.T) {
	url := newWSEchoServer(t)
	out, err := runCLI(t, "websocket", url, "-m", "hello", "--listen", "500ms")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "echo:hello")
}

func TestWebSocketAlias(t *testing.T) {
	url := newWSEchoServer(t)
	out, err := runCLI(t, "ws", url, "-m", "hi", "--listen", "500ms")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "echo:hi")
}

func TestWebSocketHeaders(t *testing.T) {
	url := newWSEchoServer(t)
	out, err := runCLI(t, "websocket", url, "-m", "x", "--listen", "500ms", "-H", "X-Token: abc")
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "token=abc")
}

func TestWebSocketInvalidHeader(t *testing.T) {
	url := newWSEchoServer(t)
	_, err := runCLI(t, "websocket", url, "-H", "bogus")
	if err == nil {
		t.Fatal("expected error for malformed header")
	}
}

func TestWebSocketHandshakeFailure(t *testing.T) {
	url := newWSEchoServer(t)
	_, err := runCLI(t, "websocket", url+"/forbidden", "-m", "x")
	if err == nil {
		t.Fatal("expected handshake error")
	}
	mustContain(t, err.Error(), "handshake failed", "403")
}

func TestWSClosedTreatsNormalCloseAsSuccess(t *testing.T) {
	if err := wsClosed(&websocket.CloseError{Code: websocket.CloseNormalClosure}); err != nil {
		t.Errorf("normal close returned %v", err)
	}
	if err := wsClosed(&websocket.CloseError{Code: websocket.CloseGoingAway}); err != nil {
		t.Errorf("going away returned %v", err)
	}
	if err := wsClosed(&websocket.CloseError{Code: websocket.CloseAbnormalClosure}); err == nil {
		t.Error("abnormal close should be an error")
	}
}

// ---- RTSP ----

func TestRTSP(t *testing.T) {
	rec := &recorder{}
	host, port := serve(t, func(c net.Conn) {
		r := bufio.NewReader(c)
		for {
			first, err := r.ReadString('\n')
			if err != nil {
				return
			}
			rec.add(strings.TrimRight(first, "\r\n"))
			for { // headers until blank line
				h, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if h = strings.TrimRight(h, "\r\n"); h == "" {
					break
				}
				rec.add(h)
			}
			if strings.HasPrefix(first, "DESCRIBE") {
				fmt.Fprint(c, "RTSP/1.0 200 OK\r\nCSeq: 2\r\nContent-Length: 11\r\n\r\nv=0 SDPBODY")
				return
			}
			fmt.Fprint(c, "RTSP/1.0 200 OK\r\nCSeq: 1\r\nPublic: OPTIONS, DESCRIBE\r\n\r\n")
		}
	})

	url := fmt.Sprintf("rtsp://%s:%s/stream", host, port)
	out, err := runCLI(t, "rtsp", url)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, ">>> OPTIONS", ">>> DESCRIBE", "Public: OPTIONS, DESCRIBE", "v=0 SDPBODY")
	mustContain(t, rec.all(),
		"OPTIONS "+url+" RTSP/1.0",
		"DESCRIBE "+url+" RTSP/1.0",
		"Accept: application/sdp",
	)
}

func TestRTSPInvalidURL(t *testing.T) {
	if _, err := runCLI(t, "rtsp", "not a url"); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// ---- RTMP ----

func rtmpHandler(version byte) func(net.Conn) {
	return func(c net.Conn) {
		c0c1 := make([]byte, 1+1536)
		if _, err := io.ReadFull(c, c0c1); err != nil {
			return
		}
		s1 := make([]byte, 1536)
		copy(s1[8:], "server-random")
		_, _ = c.Write(append([]byte{version}, s1...))
		if version != 3 {
			return
		}
		c2 := make([]byte, 1536)
		if _, err := io.ReadFull(c, c2); err != nil {
			return
		}
		_, _ = c.Write(c0c1[1:]) // S2 echoes C1
	}
}

func TestRTMPHandshake(t *testing.T) {
	host, port := serve(t, rtmpHandler(3))
	out, err := runCLI(t, "rtmp", fmt.Sprintf("rtmp://%s:%s/live", host, port))
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "RTMP handshake OK", "version 3")
}

func TestRTMPPortFlagOverridesURL(t *testing.T) {
	host, port := serve(t, rtmpHandler(3))
	out, err := runCLI(t, "rtmp", fmt.Sprintf("rtmp://%s:9/live", host), "-p", port)
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, out, "RTMP handshake OK")
}

func TestRTMPBadVersion(t *testing.T) {
	host, port := serve(t, rtmpHandler(4))
	_, err := runCLI(t, "rtmp", fmt.Sprintf("rtmp://%s:%s/live", host, port))
	if err == nil {
		t.Fatal("expected version error")
	}
	mustContain(t, err.Error(), "unexpected RTMP version 4")
}

func TestRTMPInvalidURL(t *testing.T) {
	if _, err := runCLI(t, "rtmp", "nothing"); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

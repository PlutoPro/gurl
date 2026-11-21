package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
)

func TestGopher(t *testing.T) {
	rec := &recorder{}
	host, port := serve(t, func(c net.Conn) {
		line, _ := bufio.NewReader(c).ReadString('\n')
		rec.add(strings.TrimRight(line, "\r\n"))
		fmt.Fprint(c, "iWelcome\t\terror.host\t1\r\n.\r\n")
	})

	t.Run("selector sent", func(t *testing.T) {
		out, err := runCLI(t, "gopher", host, "/news", "-p", port)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "Welcome")
		mustContain(t, rec.all(), "/news")
	})
	t.Run("empty selector", func(t *testing.T) {
		rec.lines = nil
		if _, err := runCLI(t, "gopher", host, "-p", port); err != nil {
			t.Fatal(err)
		}
		if got := rec.all(); got != "" { // CRLF only -> empty line
			t.Errorf("expected empty selector line, got %q", got)
		}
	})
}

func dictHandler(rec *recorder) func(net.Conn) {
	return func(c net.Conn) {
		r := bufio.NewReader(c)
		fmt.Fprint(c, "220 dict ready\r\n")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			rec.add(line)
			switch {
			case strings.HasPrefix(line, "DEFINE"):
				fmt.Fprint(c, "150 1 definitions\r\n151 \"go\" db\r\nA programming language.\r\n.\r\n250 ok\r\n")
			case strings.HasPrefix(line, "MATCH"):
				fmt.Fprint(c, "152 1 matches\r\ndb \"golang\"\r\n.\r\n250 ok\r\n")
			case line == "QUIT":
				fmt.Fprint(c, "221 bye\r\n")
				return
			}
		}
	}
}

func TestDict(t *testing.T) {
	rec := &recorder{}
	host, port := serve(t, dictHandler(rec))

	t.Run("define", func(t *testing.T) {
		out, err := runCLI(t, "dict", "define", host, "go", "-p", port)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "A programming language.")
		mustContain(t, rec.all(), `DEFINE ! "go"`)
	})
	t.Run("match with db", func(t *testing.T) {
		out, err := runCLI(t, "dict", "match", host, "gol", "-p", port, "--db", "*")
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "golang")
		mustContain(t, rec.all(), `MATCH * prefix "gol"`)
	})
}

// fakeConn is an in-memory io.ReadWriter for exercising telnetCopy.
type fakeConn struct {
	in  *bytes.Reader
	out bytes.Buffer
}

func (f *fakeConn) Read(p []byte) (int, error)  { return f.in.Read(p) }
func (f *fakeConn) Write(p []byte) (int, error) { return f.out.Write(p) }

func TestTelnetNegotiationRefused(t *testing.T) {
	const (
		iac, dont, do, wont, will = 255, 254, 253, 252, 251
	)
	// IAC DO 1, text, IAC WILL 3, text, escaped IAC IAC
	input := []byte{iac, do, 1, 'h', 'i', iac, will, 3, '!', iac, iac}
	conn := &fakeConn{in: bytes.NewReader(input)}
	var shown bytes.Buffer
	telnetCopy(&shown, conn)

	if want := []byte("hi!\xff"); !bytes.Equal(shown.Bytes(), want) {
		t.Errorf("displayed %q, want %q", shown.Bytes(), want)
	}
	wantReplies := []byte{iac, wont, 1, iac, dont, 3}
	if !bytes.Equal(conn.out.Bytes(), wantReplies) {
		t.Errorf("replies %v, want %v", conn.out.Bytes(), wantReplies)
	}
}

func TestTelnetPlainText(t *testing.T) {
	conn := &fakeConn{in: bytes.NewReader([]byte("login: "))}
	var shown bytes.Buffer
	telnetCopy(&shown, conn)
	if shown.String() != "login: " {
		t.Errorf("got %q", shown.String())
	}
	if conn.out.Len() != 0 {
		t.Errorf("unexpected replies: %v", conn.out.Bytes())
	}
}

var _ io.ReadWriter = (*fakeConn)(nil)

package cmd

import (
	"bytes"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// resetFlags restores every flag in the command tree to its default, because
// cobra keeps flag values in package-level variables between Execute calls.
func resetFlags(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		if sv, ok := f.Value.(pflag.SliceValue); ok {
			_ = sv.Replace(nil)
		} else {
			_ = f.Value.Set(f.DefValue)
		}
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

// runCLI executes the root command with args and returns captured stdout.
// Tests using it must not run in parallel (it swaps os.Stdout).
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags(rootCmd)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	captured := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		captured <- buf.String()
	}()

	rootCmd.SetArgs(args)
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)
	execErr := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = old
	return <-captured, execErr
}

// serve starts a TCP server on a free local port; handler runs once per
// connection. It returns the host and port to pass to a command.
func serve(t *testing.T, handler func(c net.Conn)) (host, port string) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))
				handler(c)
			}()
		}
	}()
	host, port, _ = net.SplitHostPort(l.Addr().String())
	return host, port
}

// recorder collects lines a fake server received, safely across goroutines.
type recorder struct {
	mu    sync.Mutex
	lines []string
}

func (r *recorder) add(s string) {
	r.mu.Lock()
	r.lines = append(r.lines, s)
	r.mu.Unlock()
}

func (r *recorder) all() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return strings.Join(r.lines, "\n")
}

func mustContain(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			t.Errorf("expected output to contain %q, got:\n%s", n, haystack)
		}
	}
}

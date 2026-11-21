package cmd

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

var (
	wsHeaders  []string
	wsMessage  string
	wsTimeout  time.Duration
	wsInsecure bool
	wsListen   time.Duration
)

var websocketCmd = &cobra.Command{
	Use:     "websocket <ws[s]://url>",
	Aliases: []string{"ws"},
	Short:   "Connect to a WebSocket; send --message or chat interactively",
	Example: `  gurl websocket wss://echo.websocket.org --message hello --listen 2s
  gurl websocket ws://localhost:8080/chat        # interactive, Ctrl+C to quit`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		h := http.Header{}
		for _, line := range wsHeaders {
			k, v, ok := strings.Cut(line, ":")
			if !ok {
				return fmt.Errorf("invalid header %q (expected Key: Value)", line)
			}
			h.Add(strings.TrimSpace(k), strings.TrimSpace(v))
		}
		d := websocket.Dialer{
			HandshakeTimeout: wsTimeout,
			TLSClientConfig:  &tls.Config{InsecureSkipVerify: wsInsecure},
		}
		conn, resp, err := d.Dial(args[0], h)
		if err != nil {
			if resp != nil {
				return fmt.Errorf("handshake failed: %s: %w", resp.Status, err)
			}
			return err
		}
		defer conn.Close()
		fmt.Fprintf(os.Stderr, "Connected (%s)\n", resp.Status)

		readErr := make(chan error, 1)
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					readErr <- err
					return
				}
				fmt.Println(string(msg))
			}
		}()

		if wsMessage != "" {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(wsMessage)); err != nil {
				return err
			}
			select {
			case err := <-readErr:
				return wsClosed(err)
			case <-time.After(wsListen):
				_ = conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return nil
			}
		}

		// Interactive: each stdin line is sent as a text message.
		go func() {
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				if err := conn.WriteMessage(websocket.TextMessage, sc.Bytes()); err != nil {
					return
				}
			}
			_ = conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		}()
		return wsClosed(<-readErr)
	},
}

// wsClosed treats a normal close as success.
func wsClosed(err error) error {
	if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
		return nil
	}
	return err
}

func init() {
	rootCmd.AddCommand(websocketCmd)
	f := websocketCmd.Flags()
	f.StringArrayVarP(&wsHeaders, "header", "H", nil, "Custom handshake headers")
	f.StringVarP(&wsMessage, "message", "m", "", "Send one message, wait --listen, then close")
	f.DurationVar(&wsListen, "listen", 3*time.Second, "How long to wait for replies after --message")
	f.DurationVar(&wsTimeout, "timeout", 10*time.Second, "Handshake timeout")
	f.BoolVar(&wsInsecure, "insecure", false, "Skip TLS certificate verification")
}

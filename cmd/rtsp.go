package cmd

import (
	"bufio"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

var rtspOpts netOpts

var rtspCmd = &cobra.Command{
	Use:   "rtsp <rtsp://host[:port]/path>",
	Short: "Probe an RTSP server (OPTIONS + DESCRIBE)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u, err := url.Parse(args[0])
		if err != nil || u.Host == "" {
			return fmt.Errorf("invalid RTSP url %q", args[0])
		}
		if u.Scheme == "rtsps" {
			rtspOpts.useTLS = true
		}
		if !cmd.Flags().Changed("port") {
			rtspOpts.port = 554
			if p := u.Port(); p != "" {
				fmt.Sscanf(p, "%d", &rtspOpts.port)
			}
		}
		conn, err := rtspOpts.dial(u.Hostname())
		if err != nil {
			return err
		}
		defer conn.Close()
		r := bufio.NewReader(conn)

		cseq := 0
		roundTrip := func(method, accept string) error {
			cseq++
			req := fmt.Sprintf("%s %s RTSP/1.0\r\nCSeq: %d\r\nUser-Agent: gurl\r\n", method, args[0], cseq)
			if accept != "" {
				req += "Accept: " + accept + "\r\n"
			}
			rtspOpts.deadline(conn)
			if _, err := fmt.Fprint(conn, req+"\r\n"); err != nil {
				return err
			}
			fmt.Printf(">>> %s\n", method)
			length := 0
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					return err
				}
				line = strings.TrimRight(line, "\r\n")
				if line == "" {
					break
				}
				fmt.Println(line)
				if k, v, ok := strings.Cut(line, ":"); ok && strings.EqualFold(k, "Content-Length") {
					fmt.Sscanf(strings.TrimSpace(v), "%d", &length)
				}
			}
			if length > 0 {
				body := make([]byte, length)
				if _, err := readFull(r, body); err != nil {
					return err
				}
				fmt.Printf("\n%s\n", body)
			}
			fmt.Println()
			return nil
		}

		if err := roundTrip("OPTIONS", ""); err != nil {
			return err
		}
		return roundTrip("DESCRIBE", "application/sdp")
	},
}

func readFull(r *bufio.Reader, p []byte) (int, error) {
	n := 0
	for n < len(p) {
		m, err := r.Read(p[n:])
		n += m
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func init() {
	rootCmd.AddCommand(rtspCmd)
	addNetFlags(rtspCmd, &rtspOpts, 554)
}

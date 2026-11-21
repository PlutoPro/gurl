package cmd

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/url"

	"github.com/spf13/cobra"
)

var rtmpOpts netOpts

// rtmpCmd only performs the RTMP handshake (C0/C1 -> S0/S1/S2 -> C2).
// It verifies that a server speaks RTMP; it does not publish or play streams.
var rtmpCmd = &cobra.Command{
	Use:   "rtmp <rtmp://host[:port]/app>",
	Short: "Check that a server speaks RTMP (handshake only)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u, err := url.Parse(args[0])
		if err != nil || u.Host == "" {
			return fmt.Errorf("invalid RTMP url %q", args[0])
		}
		if u.Scheme == "rtmps" {
			rtmpOpts.useTLS = true
		}
		if !cmd.Flags().Changed("port") {
			rtmpOpts.port = 1935
			if p := u.Port(); p != "" {
				fmt.Sscanf(p, "%d", &rtmpOpts.port)
			}
		}
		conn, err := rtmpOpts.dial(u.Hostname())
		if err != nil {
			return err
		}
		defer conn.Close()
		rtmpOpts.deadline(conn)

		// C0 (version 3) + C1 (4B time, 4B zero, 1528B random)
		c1 := make([]byte, 1536)
		if _, err := rand.Read(c1[8:]); err != nil {
			return err
		}
		if _, err := conn.Write(append([]byte{3}, c1...)); err != nil {
			return err
		}

		s0 := make([]byte, 1)
		if _, err := io.ReadFull(conn, s0); err != nil {
			return fmt.Errorf("reading S0: %w", err)
		}
		if s0[0] != 3 {
			return fmt.Errorf("unexpected RTMP version %d", s0[0])
		}
		s1 := make([]byte, 1536)
		if _, err := io.ReadFull(conn, s1); err != nil {
			return fmt.Errorf("reading S1: %w", err)
		}
		if _, err := conn.Write(s1); err != nil { // C2 echoes S1
			return err
		}
		s2 := make([]byte, 1536)
		if _, err := io.ReadFull(conn, s2); err != nil {
			return fmt.Errorf("reading S2: %w", err)
		}
		fmt.Printf("RTMP handshake OK with %s (version %d)\n", conn.RemoteAddr(), s0[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rtmpCmd)
	addNetFlags(rtmpCmd, &rtmpOpts, 1935)
}

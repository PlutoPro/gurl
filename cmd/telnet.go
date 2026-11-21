package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

var telnetOpts netOpts

var telnetCmd = &cobra.Command{
	Use:   "telnet <host>",
	Short: "Interactive raw TCP/telnet session (stdin <-> server)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := telnetOpts.dial(args[0])
		if err != nil {
			return err
		}
		defer conn.Close()

		done := make(chan struct{})
		go func() {
			defer close(done)
			telnetCopy(os.Stdout, conn)
		}()
		go func() {
			_, _ = io.Copy(conn, os.Stdin)
			// stdin closed: let server finish replying
		}()
		<-done
		return nil
	},
}

// telnetCopy copies server output to w, answering option negotiation with refusals.
func telnetCopy(w io.Writer, conn io.ReadWriter) {
	const (
		iac  = 255
		dont = 254
		do   = 253
		wont = 252
		will = 251
	)
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		out := make([]byte, 0, n)
		for i := 0; i < n; i++ {
			if buf[i] != iac {
				out = append(out, buf[i])
				continue
			}
			if i+1 < n && buf[i+1] == iac { // escaped 0xFF
				out = append(out, iac)
				i++
				continue
			}
			if i+2 < n {
				switch buf[i+1] {
				case do:
					_, _ = conn.Write([]byte{iac, wont, buf[i+2]})
				case will:
					_, _ = conn.Write([]byte{iac, dont, buf[i+2]})
				}
				i += 2
			}
		}
		_, _ = w.Write(out)
		if err != nil {
			return
		}
	}
}

func init() {
	rootCmd.AddCommand(telnetCmd)
	addNetFlags(telnetCmd, &telnetOpts, 23)
	telnetCmd.PersistentFlags().Lookup("timeout").Value.Set("30s")
}

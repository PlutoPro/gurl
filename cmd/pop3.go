package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/textproto"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	pop3Opts           netOpts
	pop3User, pop3Pass string
	pop3Retr           int
	pop3Stat           bool
)

var pop3Cmd = &cobra.Command{
	Use:   "pop3 <host>",
	Short: "Read mail over POP3 (list messages by default)",
	Example: `  gurl pop3 pop.example.com --tls --user me --pass secret
  gurl pop3 pop.example.com --tls --user me --pass secret --retr 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := pop3Opts.dial(args[0])
		if err != nil {
			return err
		}
		defer conn.Close()
		pop3Opts.deadline(conn)
		tp := textproto.NewConn(conn)

		// expect reads a status line and fails on -ERR.
		expect := func() (string, error) {
			line, err := tp.ReadLine()
			if err != nil {
				return "", err
			}
			if !strings.HasPrefix(line, "+OK") {
				return line, fmt.Errorf("pop3: %s", line)
			}
			return line, nil
		}
		send := func(format string, a ...any) (string, error) {
			if err := tp.PrintfLine(format, a...); err != nil {
				return "", err
			}
			return expect()
		}

		if _, err := expect(); err != nil { // greeting
			return err
		}
		if pop3User != "" {
			if _, err := send("USER %s", pop3User); err != nil {
				return err
			}
			if _, err := send("PASS %s", pop3Pass); err != nil {
				return fmt.Errorf("login failed: %w", err)
			}
		}

		switch {
		case pop3Stat:
			line, err := send("STAT")
			if err != nil {
				return err
			}
			fmt.Println(line)
		case pop3Retr > 0:
			if _, err := send("RETR %d", pop3Retr); err != nil {
				return err
			}
			if _, err := io.Copy(os.Stdout, tp.DotReader()); err != nil {
				return err
			}
		default:
			if _, err := send("LIST"); err != nil {
				return err
			}
			sc := bufio.NewScanner(tp.DotReader())
			for sc.Scan() {
				fmt.Println(sc.Text())
			}
		}
		_ = tp.PrintfLine("QUIT")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pop3Cmd)
	addNetFlags(pop3Cmd, &pop3Opts, 110)
	f := pop3Cmd.Flags()
	f.StringVar(&pop3User, "user", "", "Username")
	f.StringVar(&pop3Pass, "pass", "", "Password")
	f.IntVar(&pop3Retr, "retr", 0, "Retrieve and print message number N")
	f.BoolVar(&pop3Stat, "stat", false, "Show mailbox message count and size")
}

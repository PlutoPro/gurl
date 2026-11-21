package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	imapOpts           netOpts
	imapUser, imapPass string
	imapMailbox        string
	imapFetch          int
	imapMailboxes      bool
)

var imapCmd = &cobra.Command{
	Use:   "imap <host>",
	Short: "Read mail over IMAP (summarize a mailbox by default)",
	Example: `  gurl imap imap.example.com --tls --user me --pass secret --mailboxes
  gurl imap imap.example.com --tls --user me --pass secret --mailbox INBOX
  gurl imap imap.example.com --tls --user me --pass secret --fetch 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if imapUser == "" {
			return fmt.Errorf("--user is required")
		}
		conn, err := imapOpts.dial(args[0])
		if err != nil {
			return err
		}
		defer conn.Close()
		imapOpts.deadline(conn)
		r := bufio.NewReader(conn)

		if _, err := r.ReadString('\n'); err != nil { // greeting
			return err
		}

		n := 0
		// run sends a tagged command and prints/returns untagged lines until completion.
		run := func(print bool, format string, a ...any) error {
			n++
			tag := fmt.Sprintf("a%d", n)
			if _, err := fmt.Fprintf(conn, "%s %s\r\n", tag, fmt.Sprintf(format, a...)); err != nil {
				return err
			}
			for {
				line, err := r.ReadString('\n')
				if err != nil {
					return err
				}
				if strings.HasPrefix(line, tag+" ") {
					if !strings.HasPrefix(line[len(tag)+1:], "OK") {
						return fmt.Errorf("imap: %s", strings.TrimSpace(line))
					}
					return nil
				}
				if print {
					fmt.Print(line)
				}
			}
		}

		if err := run(false, "LOGIN %s %s", imapQuote(imapUser), imapQuote(imapPass)); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
		defer run(false, "LOGOUT")

		switch {
		case imapMailboxes:
			return run(true, `LIST "" "*"`)
		case imapFetch > 0:
			if err := run(false, "SELECT %s", imapQuote(imapMailbox)); err != nil {
				return err
			}
			return run(true, "FETCH %d BODY.PEEK[]", imapFetch)
		default:
			if err := run(false, "SELECT %s", imapQuote(imapMailbox)); err != nil {
				return err
			}
			return run(true, "FETCH 1:* (FLAGS BODY.PEEK[HEADER.FIELDS (FROM SUBJECT DATE)])")
		}
	},
}

// imapQuote returns s as an IMAP quoted string.
func imapQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func init() {
	rootCmd.AddCommand(imapCmd)
	addNetFlags(imapCmd, &imapOpts, 143)
	f := imapCmd.Flags()
	f.StringVar(&imapUser, "user", "", "Username")
	f.StringVar(&imapPass, "pass", "", "Password")
	f.StringVar(&imapMailbox, "mailbox", "INBOX", "Mailbox to open")
	f.IntVar(&imapFetch, "fetch", 0, "Fetch full message number N")
	f.BoolVar(&imapMailboxes, "mailboxes", false, "List mailboxes instead")
}

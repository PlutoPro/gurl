package cmd

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var (
	smtpOpts                        netOpts
	smtpFrom, smtpSubject, smtpBody string
	smtpTo                          []string
	smtpUser, smtpPass              string
)

var smtpCmd = &cobra.Command{
	Use:   "smtp <host>",
	Short: "Send an email via SMTP (STARTTLS used automatically when offered)",
	Example: `  gurl smtp smtp.example.com -p 587 --user me --pass secret \
      --from me@example.com --to you@example.com --subject Hi --body "Hello"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if smtpFrom == "" || len(smtpTo) == 0 {
			return fmt.Errorf("--from and at least one --to are required")
		}
		host := args[0]
		addr := net.JoinHostPort(host, strconv.Itoa(smtpOpts.port))

		conn, err := smtpOpts.dial(host)
		if err != nil {
			return err
		}
		smtpOpts.deadline(conn)
		c, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("smtp handshake with %s: %w", addr, err)
		}
		defer c.Close()

		if !smtpOpts.useTLS {
			if ok, _ := c.Extension("STARTTLS"); ok {
				if err := c.StartTLS(&tls.Config{ServerName: host, InsecureSkipVerify: smtpOpts.insecure}); err != nil {
					return fmt.Errorf("starttls: %w", err)
				}
			}
		}
		if smtpUser != "" {
			if err := c.Auth(smtp.PlainAuth("", smtpUser, smtpPass, host)); err != nil {
				return fmt.Errorf("auth: %w", err)
			}
		}
		if err := c.Mail(smtpFrom); err != nil {
			return err
		}
		for _, to := range smtpTo {
			if err := c.Rcpt(to); err != nil {
				return fmt.Errorf("rcpt %s: %w", to, err)
			}
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n",
			smtpFrom, strings.Join(smtpTo, ", "), smtpSubject, smtpBody)
		if _, err := w.Write([]byte(msg)); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		fmt.Println("Message sent")
		return c.Quit()
	},
}

func init() {
	rootCmd.AddCommand(smtpCmd)
	addNetFlags(smtpCmd, &smtpOpts, 587)
	f := smtpCmd.Flags()
	f.StringVar(&smtpFrom, "from", "", "Sender address")
	f.StringArrayVar(&smtpTo, "to", nil, "Recipient address (repeatable)")
	f.StringVar(&smtpSubject, "subject", "", "Subject line")
	f.StringVar(&smtpBody, "body", "", "Message body")
	f.StringVar(&smtpUser, "user", "", "Username for AUTH PLAIN")
	f.StringVar(&smtpPass, "pass", "", "Password for AUTH PLAIN")
}

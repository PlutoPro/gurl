package cmd

import (
	"bufio"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"testing"
)

// ---- SMTP ----

func smtpHandler(rec *recorder) func(net.Conn) {
	return func(c net.Conn) {
		tp := textproto.NewConn(c)
		_ = tp.PrintfLine("220 fake ESMTP")
		for {
			line, err := tp.ReadLine()
			if err != nil {
				return
			}
			rec.add(line)
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				_ = tp.PrintfLine("250 fake")
			case strings.HasPrefix(line, "MAIL"), strings.HasPrefix(line, "RCPT"):
				_ = tp.PrintfLine("250 OK")
			case line == "DATA":
				_ = tp.PrintfLine("354 go ahead")
				data, _ := tp.ReadDotLines()
				rec.add("DATA:" + strings.Join(data, "|"))
				_ = tp.PrintfLine("250 queued")
			case line == "QUIT":
				_ = tp.PrintfLine("221 bye")
				return
			default:
				_ = tp.PrintfLine("500 unknown")
			}
		}
	}
}

func TestSMTPSend(t *testing.T) {
	rec := &recorder{}
	host, port := serve(t, smtpHandler(rec))
	out, err := runCLI(t, "smtp", host, "-p", port,
		"--from", "me@x.com", "--to", "you@y.com", "--to", "them@z.com",
		"--subject", "Hi", "--body", "Hello there")
	if err != nil {
		t.Fatalf("smtp failed: %v", err)
	}
	mustContain(t, out, "Message sent")
	mustContain(t, rec.all(),
		"MAIL FROM:<me@x.com>",
		"RCPT TO:<you@y.com>",
		"RCPT TO:<them@z.com>",
		"Subject: Hi",
		"Hello there",
	)
}

func TestSMTPRequiresFromAndTo(t *testing.T) {
	if _, err := runCLI(t, "smtp", "localhost"); err == nil {
		t.Fatal("expected error without --from/--to")
	}
	if _, err := runCLI(t, "smtp", "localhost", "--from", "a@b.c"); err == nil {
		t.Fatal("expected error without --to")
	}
}

// ---- POP3 ----

func pop3Handler(c net.Conn) {
	tp := textproto.NewConn(c)
	_ = tp.PrintfLine("+OK ready")
	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}
		switch {
		case strings.HasPrefix(line, "USER"):
			_ = tp.PrintfLine("+OK")
		case line == "PASS bad":
			_ = tp.PrintfLine("-ERR invalid password")
		case strings.HasPrefix(line, "PASS"):
			_ = tp.PrintfLine("+OK logged in")
		case line == "STAT":
			_ = tp.PrintfLine("+OK 2 460")
		case line == "LIST":
			_ = tp.PrintfLine("+OK 2 messages")
			_ = tp.PrintfLine("1 120")
			_ = tp.PrintfLine("2 340")
			_ = tp.PrintfLine(".")
		case line == "RETR 1":
			_ = tp.PrintfLine("+OK")
			_ = tp.PrintfLine("Subject: hi")
			_ = tp.PrintfLine("")
			_ = tp.PrintfLine("body text")
			_ = tp.PrintfLine(".")
		case line == "QUIT":
			_ = tp.PrintfLine("+OK bye")
			return
		default:
			_ = tp.PrintfLine("-ERR unknown")
		}
	}
}

func TestPOP3(t *testing.T) {
	host, port := serve(t, pop3Handler)
	base := []string{"pop3", host, "-p", port, "--user", "u", "--pass", "p"}

	t.Run("list", func(t *testing.T) {
		out, err := runCLI(t, base...)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "1 120", "2 340")
	})
	t.Run("stat", func(t *testing.T) {
		out, err := runCLI(t, append(base, "--stat")...)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "+OK 2 460")
	})
	t.Run("retr", func(t *testing.T) {
		out, err := runCLI(t, append(base, "--retr", "1")...)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "Subject: hi", "body text")
	})
	t.Run("bad password", func(t *testing.T) {
		_, err := runCLI(t, "pop3", host, "-p", port, "--user", "u", "--pass", "bad")
		if err == nil {
			t.Fatal("expected login failure")
		}
		mustContain(t, err.Error(), "login failed")
	})
}

// ---- IMAP ----

func imapHandler(c net.Conn) {
	r := bufio.NewReader(c)
	fmt.Fprint(c, "* OK ready\r\n")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		f := strings.SplitN(strings.TrimSpace(line), " ", 3)
		if len(f) < 2 {
			return
		}
		tag, cmd, rest := f[0], f[1], ""
		if len(f) == 3 {
			rest = f[2]
		}
		switch cmd {
		case "LOGIN":
			if strings.Contains(rest, `"bad"`) {
				fmt.Fprintf(c, "%s NO invalid credentials\r\n", tag)
			} else {
				fmt.Fprintf(c, "%s OK logged in\r\n", tag)
			}
		case "SELECT":
			fmt.Fprintf(c, "* 2 EXISTS\r\n%s OK selected\r\n", tag)
		case "LIST":
			fmt.Fprintf(c, "* LIST () \".\" INBOX\r\n* LIST () \".\" Sent\r\n%s OK done\r\n", tag)
		case "FETCH":
			fmt.Fprintf(c, "* 1 FETCH (BODY[] {5}\r\nhello)\r\n%s OK done\r\n", tag)
		case "LOGOUT":
			fmt.Fprintf(c, "* BYE\r\n%s OK bye\r\n", tag)
			return
		default:
			fmt.Fprintf(c, "%s BAD unknown\r\n", tag)
		}
	}
}

func TestIMAP(t *testing.T) {
	host, port := serve(t, imapHandler)
	base := []string{"imap", host, "-p", port, "--user", "u", "--pass", "p"}

	t.Run("mailboxes", func(t *testing.T) {
		out, err := runCLI(t, append(base, "--mailboxes")...)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "INBOX", "Sent")
	})
	t.Run("summary", func(t *testing.T) {
		out, err := runCLI(t, base...)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "FETCH")
	})
	t.Run("fetch", func(t *testing.T) {
		out, err := runCLI(t, append(base, "--fetch", "1")...)
		if err != nil {
			t.Fatal(err)
		}
		mustContain(t, out, "hello")
	})
	t.Run("bad login", func(t *testing.T) {
		_, err := runCLI(t, "imap", host, "-p", port, "--user", "u", "--pass", "bad")
		if err == nil {
			t.Fatal("expected login failure")
		}
		mustContain(t, err.Error(), "login failed")
	})
	t.Run("requires user", func(t *testing.T) {
		if _, err := runCLI(t, "imap", host, "-p", port); err == nil {
			t.Fatal("expected error without --user")
		}
	})
}

func TestIMAPQuote(t *testing.T) {
	tests := map[string]string{
		`plain`:      `"plain"`,
		`has "q"`:    `"has \"q\""`,
		`back\slash`: `"back\\slash"`,
	}
	for in, want := range tests {
		if got := imapQuote(in); got != want {
			t.Errorf("imapQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

package cmd

import (
	"testing"
)

// Argument/flag validation and unreachable-target behaviour for the commands
// that need real third-party servers (SSH, LDAP, MQTT) to do anything useful.

func TestSSHRequiresAuth(t *testing.T) {
	_, err := runCLI(t, "ssh", "user@127.0.0.1", "uptime", "--insecure")
	if err == nil {
		t.Fatal("expected error without --key/--pass")
	}
	mustContain(t, err.Error(), "provide --key or --pass")
}

func TestSSHRequiresCommand(t *testing.T) {
	if _, err := runCLI(t, "ssh", "user@127.0.0.1"); err == nil {
		t.Fatal("expected arg-count error")
	}
}

func TestSSHMissingKeyFile(t *testing.T) {
	_, err := runCLI(t, "ssh", "u@127.0.0.1", "ls", "--key", "does-not-exist.pem")
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
}

func TestSSHUnreachable(t *testing.T) {
	_, err := runCLI(t, "ssh", "u@127.0.0.1", "ls", "-p", "1", "--pass", "x", "--insecure", "--timeout", "1s")
	if err == nil {
		t.Fatal("expected connection error")
	}
	mustContain(t, err.Error(), "ssh connect")
}

func TestLDAPRequiresBase(t *testing.T) {
	_, err := runCLI(t, "ldap", "ldap://127.0.0.1:1")
	if err == nil {
		t.Fatal("expected error without --base")
	}
	mustContain(t, err.Error(), "--base is required")
}

func TestLDAPUnreachable(t *testing.T) {
	_, err := runCLI(t, "ldap", "ldap://127.0.0.1:1", "--base", "dc=x", "--timeout", "1s")
	if err == nil {
		t.Fatal("expected connection error")
	}
	mustContain(t, err.Error(), "connect")
}

func TestMQTTUnreachableBroker(t *testing.T) {
	_, err := runCLI(t, "mqtt", "pub", "a/b", "msg", "-b", "tcp://127.0.0.1:1", "--timeout", "1s")
	if err == nil {
		t.Fatal("expected connection error")
	}
	mustContain(t, err.Error(), "connect")
}

func TestMQTTPubRequiresTwoArgs(t *testing.T) {
	if _, err := runCLI(t, "mqtt", "pub", "only-topic"); err == nil {
		t.Fatal("expected arg-count error")
	}
}

func TestNetCommandsFailFastOnClosedPort(t *testing.T) {
	cases := [][]string{
		{"telnet", "127.0.0.1", "-p", "1"},
		{"gopher", "127.0.0.1", "-p", "1"},
		{"dict", "define", "127.0.0.1", "w", "-p", "1"},
		{"pop3", "127.0.0.1", "-p", "1"},
		{"imap", "127.0.0.1", "-p", "1", "--user", "u"},
		{"smtp", "127.0.0.1", "-p", "1", "--from", "a@b.c", "--to", "d@e.f"},
	}
	for _, args := range cases {
		t.Run(args[0], func(t *testing.T) {
			if _, err := runCLI(t, append(args, "--timeout", "1s")...); err == nil {
				t.Fatalf("%v: expected connection error", args)
			}
		})
	}
}

func TestEveryProtocolCommandRegistered(t *testing.T) {
	want := []string{"http", "https", "websocket", "mqtt", "ssh", "ldap", "smtp",
		"pop3", "imap", "telnet", "gopher", "dict", "rtsp", "rtmp"}
	have := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		have[c.Name()] = true
	}
	for _, name := range want {
		if !have[name] {
			t.Errorf("command %q not registered on root", name)
		}
	}
}

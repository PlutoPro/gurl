package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var (
	sshPort     int
	sshPass     string
	sshKey      string
	sshInsecure bool
	sshTimeout  time.Duration
)

var sshCmd = &cobra.Command{
	Use:   "ssh <[user@]host> <command>",
	Short: "Run a remote command over SSH",
	Example: `  gurl ssh me@host.example.com "uname -a" --key ~/.ssh/id_ed25519
  gurl ssh me@host.example.com "uptime" --pass secret`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		user, host := os.Getenv("USERNAME"), args[0]
		if u, h, ok := strings.Cut(args[0], "@"); ok {
			user, host = u, h
		}
		command := strings.Join(args[1:], " ")

		var auth []ssh.AuthMethod
		if sshKey != "" {
			pem, err := os.ReadFile(sshKey)
			if err != nil {
				return err
			}
			signer, err := ssh.ParsePrivateKey(pem)
			if err != nil {
				return fmt.Errorf("parse key (passphrase-protected keys are not supported): %w", err)
			}
			auth = append(auth, ssh.PublicKeys(signer))
		}
		if sshPass != "" {
			auth = append(auth, ssh.Password(sshPass))
		}
		if len(auth) == 0 {
			return fmt.Errorf("provide --key or --pass")
		}

		var hostKey ssh.HostKeyCallback
		if sshInsecure {
			hostKey = ssh.InsecureIgnoreHostKey()
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			hostKey, err = knownhosts.New(filepath.Join(home, ".ssh", "known_hosts"))
			if err != nil {
				return fmt.Errorf("load known_hosts (use --insecure to skip verification): %w", err)
			}
		}

		client, err := ssh.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(sshPort)), &ssh.ClientConfig{
			User:            user,
			Auth:            auth,
			HostKeyCallback: hostKey,
			Timeout:         sshTimeout,
		})
		if err != nil {
			return fmt.Errorf("ssh connect: %w", err)
		}
		defer client.Close()

		sess, err := client.NewSession()
		if err != nil {
			return err
		}
		defer sess.Close()
		sess.Stdout, sess.Stderr = os.Stdout, os.Stderr
		return sess.Run(command)
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)
	f := sshCmd.Flags()
	f.IntVarP(&sshPort, "port", "p", 22, "SSH port")
	f.StringVar(&sshPass, "pass", "", "Password")
	f.StringVarP(&sshKey, "key", "i", "", "Path to private key")
	f.BoolVar(&sshInsecure, "insecure", false, "Skip host key verification (unsafe)")
	f.DurationVar(&sshTimeout, "timeout", 10*time.Second, "Connection timeout")
}

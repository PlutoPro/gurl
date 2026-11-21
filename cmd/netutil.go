package cmd

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// netOpts holds flags shared by the raw-socket protocol commands.
type netOpts struct {
	port     int
	timeout  time.Duration
	useTLS   bool
	insecure bool
}

// addNetFlags registers the common connection flags on a command.
func addNetFlags(c *cobra.Command, o *netOpts, defaultPort int) {
	c.PersistentFlags().IntVarP(&o.port, "port", "p", defaultPort, "Server port")
	c.PersistentFlags().DurationVar(&o.timeout, "timeout", 10*time.Second, "Connection/IO timeout")
	c.PersistentFlags().BoolVar(&o.useTLS, "tls", false, "Connect using TLS")
	c.PersistentFlags().BoolVar(&o.insecure, "insecure", false, "Skip TLS certificate verification")
}

// dial opens a TCP (optionally TLS) connection to host:port.
func (o *netOpts) dial(host string) (net.Conn, error) {
	addr := net.JoinHostPort(host, strconv.Itoa(o.port))
	d := &net.Dialer{Timeout: o.timeout}
	if o.useTLS {
		return tls.DialWithDialer(d, "tcp", addr, &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: o.insecure,
		})
	}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", addr, err)
	}
	return conn, nil
}

// deadline refreshes the connection's IO deadline.
func (o *netOpts) deadline(c net.Conn) {
	_ = c.SetDeadline(time.Now().Add(o.timeout))
}

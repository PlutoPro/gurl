package cmd

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/spf13/cobra"
)

var (
	ldapBindDN, ldapPass, ldapBase, ldapFilter string
	ldapAttrs                                  []string
	ldapInsecure                               bool
	ldapTimeout                                time.Duration
	ldapLimit                                  int
)

var ldapCmd = &cobra.Command{
	Use:   "ldap <ldap[s]://host[:port]>",
	Short: "Search an LDAP directory",
	Example: `  gurl ldap ldap://ldap.example.com --base dc=example,dc=com --filter "(uid=jdoe)"
  gurl ldap ldaps://ldap.example.com --bind-dn "cn=admin,dc=example,dc=com" --pass secret \
      --base dc=example,dc=com --attrs cn --attrs mail`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if ldapBase == "" {
			return fmt.Errorf("--base is required")
		}
		conn, err := ldap.DialURL(args[0],
			ldap.DialWithDialer(&net.Dialer{Timeout: ldapTimeout}),
			ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: ldapInsecure}))
		if err != nil {
			return fmt.Errorf("connect: %w", err)
		}
		defer conn.Close()
		conn.SetTimeout(ldapTimeout)

		if ldapBindDN != "" {
			if err := conn.Bind(ldapBindDN, ldapPass); err != nil {
				return fmt.Errorf("bind: %w", err)
			}
		} else if err := conn.UnauthenticatedBind(""); err != nil {
			return fmt.Errorf("anonymous bind: %w", err)
		}

		res, err := conn.Search(ldap.NewSearchRequest(
			ldapBase, ldap.ScopeWholeSubtree, ldap.NeverDerefAliases,
			ldapLimit, int(ldapTimeout.Seconds()), false,
			ldapFilter, ldapAttrs, nil))
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}
		for _, e := range res.Entries {
			fmt.Println("dn:", e.DN)
			for _, a := range e.Attributes {
				fmt.Printf("%s: %s\n", a.Name, strings.Join(a.Values, ", "))
			}
			fmt.Println()
		}
		fmt.Printf("%d entries\n", len(res.Entries))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(ldapCmd)
	f := ldapCmd.Flags()
	f.StringVar(&ldapBindDN, "bind-dn", "", "DN to bind as (anonymous if empty)")
	f.StringVar(&ldapPass, "pass", "", "Bind password")
	f.StringVar(&ldapBase, "base", "", "Search base DN")
	f.StringVar(&ldapFilter, "filter", "(objectClass=*)", "LDAP search filter")
	f.StringArrayVar(&ldapAttrs, "attrs", nil, "Attributes to return (repeatable; default all)")
	f.IntVar(&ldapLimit, "limit", 0, "Max entries (0 = server default)")
	f.BoolVar(&ldapInsecure, "insecure", false, "Skip TLS certificate verification")
	f.DurationVar(&ldapTimeout, "timeout", 10*time.Second, "Connect/search timeout")
}

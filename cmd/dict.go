package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	dictOpts netOpts
	dictDB   string
)

var dictCmd = &cobra.Command{
	Use:   "dict",
	Short: "Query a DICT (RFC 2229) dictionary server",
}

func dictQuery(host, command string) error {
	conn, err := dictOpts.dial(host)
	if err != nil {
		return err
	}
	defer conn.Close()
	dictOpts.deadline(conn)
	if _, err := fmt.Fprintf(conn, "%s\r\nQUIT\r\n", command); err != nil {
		return err
	}
	_, err = io.Copy(os.Stdout, conn)
	return err
}

var dictDefineCmd = &cobra.Command{
	Use:   "define <host> <word>",
	Short: "Look up the definition of a word",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return dictQuery(args[0], fmt.Sprintf("DEFINE %s %q", dictDB, args[1]))
	},
}

var dictMatchCmd = &cobra.Command{
	Use:   "match <host> <word>",
	Short: "Find words matching a prefix",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return dictQuery(args[0], fmt.Sprintf("MATCH %s prefix %q", dictDB, args[1]))
	},
}

func init() {
	rootCmd.AddCommand(dictCmd)
	addNetFlags(dictCmd, &dictOpts, 2628)
	dictCmd.PersistentFlags().StringVar(&dictDB, "db", "!", "Dictionary database (! = first match, * = all)")
	dictCmd.AddCommand(dictDefineCmd, dictMatchCmd)
}

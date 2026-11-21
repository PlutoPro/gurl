package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

var gopherOpts netOpts

var gopherCmd = &cobra.Command{
	Use:   "gopher <host> [selector]",
	Short: "Fetch a Gopher menu or document",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		selector := ""
		if len(args) == 2 {
			selector = args[1]
		}
		conn, err := gopherOpts.dial(args[0])
		if err != nil {
			return err
		}
		defer conn.Close()
		gopherOpts.deadline(conn)
		if _, err := io.WriteString(conn, selector+"\r\n"); err != nil {
			return err
		}
		_, err = io.Copy(os.Stdout, conn)
		return err
	},
}

func init() {
	rootCmd.AddCommand(gopherCmd)
	addNetFlags(gopherCmd, &gopherOpts, 70)
}

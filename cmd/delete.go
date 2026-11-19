package cmd

import (
	nethttp "net/http"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <url>",
	Short: "Send an HTTP DELETE request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendRequest(nethttp.MethodDelete, args[0], "")
	},
}

func init() {
	httpCmd.AddCommand(deleteCmd)
}

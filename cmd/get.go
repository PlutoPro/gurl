package cmd

import (
	nethttp "net/http"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <url>",
	Short: "Send an HTTP GET request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendRequest(nethttp.MethodGet, args[0], "")
	},
}

func init() {
	httpCmd.AddCommand(getCmd)
}

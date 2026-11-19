package cmd

import (
	nethttp "net/http"

	"github.com/spf13/cobra"
)

var putBody string

var putCmd = &cobra.Command{
	Use:   "put <url>",
	Short: "Send an HTTP PUT request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendRequest(nethttp.MethodPut, args[0], putBody)
	},
}

func init() {
	httpCmd.AddCommand(putCmd)
	putCmd.Flags().StringVarP(&putBody, "data", "d", "", "Request body payload")
}

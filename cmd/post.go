package cmd

import (
	nethttp "net/http"

	"github.com/spf13/cobra"
)

var postBody string

var postCmd = &cobra.Command{
	Use:   "post <url>",
	Short: "Send an HTTP POST request",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendRequest(nethttp.MethodPost, args[0], postBody)
	},
}

func init() {
	httpCmd.AddCommand(postCmd)
	postCmd.Flags().StringVarP(&postBody, "data", "d", "", "Request body payload")
}

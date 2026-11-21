package cmd

import (
	nethttp "net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var httpsCmd = &cobra.Command{
	Use:   "https <url>",
	Short: "Send an HTTPS GET request (scheme defaults to https://)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u := args[0]
		if !strings.Contains(u, "://") {
			u = "https://" + u
		}
		return sendRequest(nethttp.MethodGet, u, "")
	},
}

func init() {
	rootCmd.AddCommand(httpsCmd)
	httpsCmd.Flags().StringArrayVarP(&headers, "header", "H", nil, "Custom headers")
	httpsCmd.Flags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "Request timeout")
}

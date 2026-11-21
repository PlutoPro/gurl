package cmd

import "github.com/spf13/cobra"

var httpsCmd = &cobra.Command{
	Use:   "https",
	Short: "Send HTTPS requests (scheme defaults to https://)",
	Long:  "Make GET, POST, PUT, and DELETE requests over HTTPS. A URL without a scheme is treated as https://.",
}

func init() {
	rootCmd.AddCommand(httpsCmd)
	addHTTPFlags(httpsCmd)
	addHTTPVerbs(httpsCmd, "https")
}

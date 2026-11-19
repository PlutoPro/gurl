/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "Gurl",
	Short: "Gurl is a command-line web client for making requests to servers. It is inspired by curl and provides similar functionality, but with a focus on simplicity and ease of use.",
	Long: `Gurl is a command-line web client for making requests to servers. It is inspired by curl and provides similar functionality, but with a focus on simplicity and ease of use.
It supports various HTTP methods, including GET, POST, PUT, and DELETE, and allows you to customize requests with headers, query parameters, and request bodies. Gurl is designed to be easy to use and understand, making it a great tool for developers and system administrators who need to interact with web services from the command line.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.Gurl.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

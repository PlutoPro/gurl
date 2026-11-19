// cmd/http.go
package cmd

import (
	"fmt"
	"io"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	headers []string
	timeout time.Duration
)

var httpCmd = &cobra.Command{
	Use:   "http",
	Short: "Send HTTP requests",
	Long:  "Make GET, POST, PUT, and DELETE HTTP requests from the CLI.",
}

func init() {
	rootCmd.AddCommand(httpCmd)

	httpCmd.PersistentFlags().StringArrayVarP(&headers, "header", "H", nil, "Custom headers (e.g. 'Authorization: Bearer token')")
	httpCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "Request timeout")
}

// sendRequest executes an HTTP request and prints the status and response body.
func sendRequest(method, rawURL, rawBody string) error {
	var bodyReader io.Reader
	if rawBody != "" {
		bodyReader = strings.NewReader(rawBody)
	}

	req, err := nethttp.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for _, h := range headers {
		key, val, ok := strings.Cut(h, ":")
		if !ok {
			return fmt.Errorf("invalid header format %q (expected Key: Value)", h)
		}
		req.Header.Add(strings.TrimSpace(key), strings.TrimSpace(val))
	}

	if rawBody != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &nethttp.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("Status: %s\n%s\n", resp.Status, string(respBody))
	return nil
}

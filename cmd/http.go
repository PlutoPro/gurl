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
	addHTTPFlags(httpCmd)
	addHTTPVerbs(httpCmd, "http")
}

// addHTTPFlags registers the flags shared by the http and https commands.
func addHTTPFlags(parent *cobra.Command) {
	parent.PersistentFlags().StringArrayVarP(&headers, "header", "H", nil, "Custom headers (e.g. 'Authorization: Bearer token')")
	parent.PersistentFlags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "Request timeout")
}

// normalizeURL prepends scheme:// when raw has no scheme.
func normalizeURL(raw, scheme string) string {
	if strings.Contains(raw, "://") {
		return raw
	}
	return scheme + "://" + raw
}

// addHTTPVerbs adds get/post/put/delete subcommands to parent. A URL given
// without a scheme gets defaultScheme prepended.
func addHTTPVerbs(parent *cobra.Command, defaultScheme string) {
	verbs := []struct {
		name, method string
		hasBody      bool
	}{
		{"get", nethttp.MethodGet, false},
		{"post", nethttp.MethodPost, true},
		{"put", nethttp.MethodPut, true},
		{"delete", nethttp.MethodDelete, false},
	}
	for _, v := range verbs {
		var body string
		c := &cobra.Command{
			Use:   v.name + " <url>",
			Short: fmt.Sprintf("Send an %s %s request", strings.ToUpper(defaultScheme), v.method),
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return sendRequest(v.method, normalizeURL(args[0], defaultScheme), body)
			},
		}
		if v.hasBody {
			c.Flags().StringVarP(&body, "data", "d", "", "Request body payload")
		}
		parent.AddCommand(c)
	}
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

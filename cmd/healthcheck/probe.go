package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// run probes the one URL in args: 0 when it answers 2xx, 1 otherwise (Docker's healthy and unhealthy). The reason
// for a failure goes to stderr, which Docker keeps in the container's health log; it names the status or the
// connection error, never a body.
func run(args []string, client *http.Client, stderr io.Writer) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: healthcheck <url>")
		return 1
	}
	if err := probe(client, args[0]); err != nil {
		_, _ = fmt.Fprintln(stderr, "healthcheck:", err)
		return 1
	}
	return 0
}

func probe(client *http.Client, target string) error {
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%q is not an absolute http(s) URL", target)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

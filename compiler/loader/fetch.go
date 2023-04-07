package loader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type loadClient struct {
	client *http.Client
}

var defaultClient = new(loadClient)

var (
	NotModified = errors.New("file not modified")
)

func (l *loadClient) newClient() *http.Client {
	cli := new(http.Client)

	t := new(http.Transport)

	t.RegisterProtocol("file", http.NewFileTransport(
		http.Dir(""),
	))

	cli.CheckRedirect = l.checkRedirect
	cli.Transport = t

	return cli

}

func (l *loadClient) checkRedirect(r *http.Request, _ []*http.Request) error {
	return fmt.Errorf("unimplemented")
}

func (l *loadClient) Fetch(ctx context.Context, netUrl *url.URL) (*bytes.Buffer, error) {

	netUrl.Fragment = ""

	if netUrl.Scheme == "" {
		netUrl.Scheme = "file"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, netUrl.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching remote file: failed to prepare request: %w", err)
	}

	client := l.newClient()

	fmt.Println("fetching", netUrl.String())
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching remote file: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status from server fetching %s: %s", netUrl.Redacted(), resp.Status)
	}

	data := new(bytes.Buffer)
	_, err = io.Copy(data, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return data, nil
}

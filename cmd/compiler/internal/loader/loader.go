package loader

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Loader interface {
	Load(context.Context, string) ([]byte, error)
}

type sourceLoader struct {
	config *sourceLoadConfig
}

type sourceLoadConfig struct {
	AllowedHosts       []string
	AllowInsecure      bool
	BlockRemote        bool
	ChrootTo           string
	AuthorizationToken string
}

var defaultConfig = &sourceLoadConfig{
	BlockRemote: true,
}

func (l *sourceLoader) Load(ctx context.Context, uri string) ([]byte, error) {

	if strings.ContainsRune(uri, ':') {
		return l.loadExternal(ctx, uri)
	}
	return l.loadFile(uri)
}

func (l *sourceLoader) loadFile(filename string) ([]byte, error) {
	var fs fs.FS
	if l.config.ChrootTo != "" {
		fs = os.DirFS(l.config.ChrootTo)
	} else {
		fs = os.DirFS(".")
	}

	file, err := fs.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filename, err)
	}

	return data, err
}

func (l *sourceLoader) loadExternal(ctx context.Context, uri string) ([]byte, error) {

	netUrl, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s: %w", uri, err)
	}

	if l.config.BlockRemote {
		return nil, fmt.Errorf("unable to load %s: loading external sources blocked", uri)
	}

	client := new(http.Client)

	if len(l.config.AllowedHosts) > 0 {
		allow := false
		for _, host := range l.config.AllowedHosts {
			if strings.EqualFold(netUrl.Hostname(), host) {
				allow = true
				break
			}
		}
		if !allow {
			return nil, fmt.Errorf("unable to load %s: loading source from %s blocked", uri, netUrl.Hostname())
		}
		client.CheckRedirect = l.checkRedirect
	}

	if netUrl.Scheme == "http" && !l.config.AllowInsecure {
		return nil, fmt.Errorf("unable to load %s: loading over plaintext http blocked", uri)
	}

	netUrl.Fragment = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, netUrl.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("error creating external request: %s", err)
	}

	if l.config.AuthorizationToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", l.config.AuthorizationToken))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching source: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error fetching %s: server returned %s", resp.Request.URL.Redacted(), resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("connection inturrupted while fetching response: %w", err)
	}

	return data, nil

}

func (l *sourceLoader) checkRedirect(req *http.Request, _ []*http.Request) error {
	allow := false
	for _, host := range l.config.AllowedHosts {
		if strings.EqualFold(req.Host, host) {
			allow = true
			break
		}
	}
	if !allow {
		return fmt.Errorf("unable to follow redurect to %s: loading source from %s blocked", req.URL.Redacted(), req.Host)
	}
	return nil
}

type loaderOption func(*sourceLoadConfig)

func RootedAt(path string) loaderOption {
	return func(conf *sourceLoadConfig) {
		conf.ChrootTo = path
	}
}

func AllowedRemoteHosts(hosts ...string) loaderOption {
	return func(conf *sourceLoadConfig) {
		conf.AllowedHosts = hosts
	}
}

func AllowInsecure() loaderOption {
	return func(conf *sourceLoadConfig) {
		conf.AllowInsecure = true
	}
}

func SetAuthorizationToken(token string) loaderOption {
	return func(conf *sourceLoadConfig) {
		conf.AuthorizationToken = token
	}
}

func NewLoader(opts ...loaderOption) Loader {
	l := new(sourceLoader)
	conf := *defaultConfig
	l.config = &conf
	for i := range opts {
		opts[i](l.config)
	}
	return l
}

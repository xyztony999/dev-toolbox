package runtimeinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// Provider hosts are fixed per the gated design; no other origin may be
// contacted, DNS answers must be public, and redirects must stay on the
// allowlist over HTTPS.
var allowedHosts = map[string]bool{
	"nodejs.org":                    true,
	"registry.npmmirror.com":        true,
	"mirrors.tuna.tsinghua.edu.cn":  true,
	"mirrors.cloud.tencent.com":     true,
	"api.adoptium.net":              true,
	"repo.maven.apache.org":         true,
	"dlcdn.apache.org":              true,
	"services.gradle.org":           true,
	"dl.google.com":                 true,
	"golang.google.cn":              true,
	"github.com":                    true,
	"objects.githubusercontent.com": true,
}

const (
	metadataTimeout = 30 * time.Second
	downloadTimeout = 20 * time.Minute
	// maxArchiveBytes bounds a single downloaded archive; maxExpandedBytes
	// and maxArchiveFiles bound what extraction may produce.
	maxArchiveBytes  int64 = 2 << 30
	maxExpandedBytes int64 = 6 << 30
	maxArchiveFiles        = 100000
)

func safeHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			if !allowedHosts[strings.ToLower(host)] {
				return nil, errors.New("RUNTIME_PROVIDER_HOST_REJECTED")
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			var last error
			for _, entry := range ips {
				if !isPublicIP(entry.IP) {
					return nil, errors.New("RUNTIME_PROVIDER_ADDRESS_REJECTED")
				}
				conn, dialErr := (&net.Dialer{Timeout: 15 * time.Second}).DialContext(ctx, network, net.JoinHostPort(entry.IP.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				last = dialErr
			}
			if last == nil {
				last = errors.New("RUNTIME_PROVIDER_ADDRESS_UNAVAILABLE")
			}
			return nil, last
		},
	}
	return &http.Client{
		Transport: transport,
		Timeout:   downloadTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || !allowedHosts[strings.ToLower(req.URL.Hostname())] || req.URL.Scheme != "https" {
				return errors.New("RUNTIME_PROVIDER_REDIRECT_REJECTED")
			}
			return nil
		},
	}
}

func isPublicIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsUnspecified() && !ip.IsMulticast()
}

// fetcher performs provider HTTP access. Tests inject rewrite plus a plain
// client so fixed provider hosts resolve to a local stub server instead of
// the network.
type fetcher struct {
	client  *http.Client
	rewrite func(string) string
}

// fetcherFor is the injection point used by the task manager and tests.
var fetcherFor = func() *fetcher { return &fetcher{client: safeHTTPClient()} }

func (f *fetcher) resolve(raw string) string {
	if f.rewrite != nil {
		return f.rewrite(raw)
	}
	return raw
}

func (f *fetcher) do(ctx context.Context, raw, method string, timeout time.Duration) (*http.Response, error) {
	target := f.resolve(raw)
	if f.rewrite == nil && !strings.HasPrefix(target, "https://") {
		return nil, errors.New("RUNTIME_PROVIDER_URL_REJECTED")
	}
	req, err := http.NewRequestWithContext(ctx, method, target, nil)
	if err != nil {
		return nil, err
	}
	client := *f.client
	client.Timeout = timeout
	return client.Do(req)
}

func (f *fetcher) getJSON(ctx context.Context, raw string, target any) error {
	resp, err := f.do(ctx, raw, http.MethodGet, metadataTimeout)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: provider HTTP %d for metadata", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (f *fetcher) getText(ctx context.Context, raw string) (string, error) {
	resp, err := f.do(ctx, raw, http.MethodGet, metadataTimeout)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: provider HTTP %d for checksum source", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (f *fetcher) head(ctx context.Context, raw string) (int64, bool) {
	resp, err := f.do(ctx, raw, http.MethodHead, metadataTimeout)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, false
	}
	return resp.ContentLength, resp.ContentLength > 0
}

func (f *fetcher) download(ctx context.Context, raw, dest string, progress func(received, total int64)) error {
	resp, err := f.do(ctx, raw, http.MethodGet, downloadTimeout)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RUNTIME_DOWNLOAD_FAILED: provider HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxArchiveBytes {
		return errors.New("RUNTIME_DOWNLOAD_FAILED: archive exceeds download size limit")
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 128*1024)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return errors.New("RUNTIME_INSTALL_CANCELLED")
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			total += int64(n)
			if total > maxArchiveBytes {
				return errors.New("RUNTIME_DOWNLOAD_FAILED: archive exceeds download size limit")
			}
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			if progress != nil {
				progress(total, resp.ContentLength)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	return out.Sync()
}

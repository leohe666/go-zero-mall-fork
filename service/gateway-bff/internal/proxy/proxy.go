// Code scaffolded for BFF API Gateway.
package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
)

type ProxyHandler struct {
	prefix string
	target string
	strip  bool
	proxy  *httputil.ReverseProxy
	mu     sync.RWMutex
}

func NewProxyHandler(prefix, target string, strip bool) (*ProxyHandler, error) {
	targetURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	
	// Customize the director to handle path stripping
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if strip {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if !strings.HasPrefix(req.URL.Path, "/") {
				req.URL.Path = "/" + req.URL.Path
			}
		}
		req.Header.Set("X-Forwarded-Prefix", prefix)
	}

	// Log errors
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logx.Errorf("proxy error for %s -> %s: %v", prefix, target, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return &ProxyHandler{
		prefix: prefix,
		target: target,
		strip:  strip,
		proxy:  proxy,
	}, nil
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logx.Infow("proxy request",
		logx.Field("prefix", p.prefix),
		logx.Field("target", p.target),
		logx.Field("method", r.Method),
		logx.Field("path", r.URL.Path),
		logx.Field("remote", r.RemoteAddr),
	)
	p.proxy.ServeHTTP(w, r)
}

// UpdateTarget allows updating the target URL at runtime
func (p *ProxyHandler) UpdateTarget(newTarget string) error {
	targetURL, err := url.Parse(newTarget)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.target = newTarget
	p.proxy = httputil.NewSingleHostReverseProxy(targetURL)
	return nil
}
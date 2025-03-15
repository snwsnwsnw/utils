package server

import (
	"context"
	"github.com/lesismal/llib/std/net/http/httputil"
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"
)

var ReverseProxy *MultiReverseProxy

type MultiReverseProxy struct {
	backends []*url.URL
	counter  uint32
}

func HandleHTTP(w http.ResponseWriter, r *http.Request) {
	if ReverseProxy == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	idx := atomic.AddUint32(&ReverseProxy.counter, 1)
	target := ReverseProxy.backends[idx%uint32(len(ReverseProxy.backends))]

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ServeHTTP(w, r)
}

func initMultiReverseProxy(ctx context.Context, backendAddress []string) {
	if ReverseProxy != nil {
		return
	}

	backends := make([]*url.URL, len(backendAddress))
	for i, s := range backendAddress {
		u, err := url.Parse("http://" + s)
		if err != nil {
			slog.ErrorContext(ctx, "[MultiReverseProxy] parse backend address error", "err", err)
			return
		}
		backends[i] = u
	}
	ReverseProxy = &MultiReverseProxy{backends: backends}
}

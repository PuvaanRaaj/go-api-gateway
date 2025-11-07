package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func PathPrefixProxy(prefix, target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		proxy.ServeHTTP(w, r)
	})
}

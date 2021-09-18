package measuredb

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/net/publicsuffix"
)

func newDefaultCompoundResolver(logger netxlite.Logger, db DB) netxlite.Resolver {
	sys := &netxlite.ResolverSystem{}
	d := WrapConnector(db, netxlite.NewConnector(logger))
	udp53 := dnsx.NewSerialResolver(
		WrapDNSRoundTripper(db, dnsx.NewDNSOverUDP(d, "8.8.4.4:53")))
	return WrapResolvers(db, sys, udp53)
}

// NewHTTPTransportStdlib is a convenience factory for creating
// a new nextlite.HTTPTransport that uses the stdlib functionality
// for resolving domain names and for TLS.
//
// Note: In addition to using the system resolver, this transport
// may also use additional resolvers.
func NewHTTPTransportStdlib(logger netxlite.Logger, db DB) netxlite.HTTPTransport {
	resolver := newDefaultCompoundResolver(logger, db)
	connector := WrapConnector(db, netxlite.NewConnector(logger))
	thx := WrapTLSHandshaker(db, netxlite.NewTLSHandshakerStdlib(logger))
	dialer := NewDialer(db, logger, resolver, connector)
	td := netxlite.NewTLSDialer(dialer, thx)
	return netxlite.WrapHTTPTransport(logger, WrapHTTPTransport(
		db, netxlite.NewOOHTTPBaseTransport(dialer, td),
	))
}

// NewCookieJar is a convenience factory for creating an http.CookieJar
// that is aware of the effective TLS / public suffix list. This
// means that the jar won't allow a domain to set cookies for another
// unrelated domain (in the public-suffix-list sense).
func NewCookieJar() http.CookieJar {
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	// Safe to assert here: cookiejar.New _always_ returns nil.
	runtimex.PanicOnError(err, "cookiejar.New failed")
	return jar
}

// NewHTTPClient is a convenience factory for creating an http.Client
// using the given HTTPTransport and with support for cookies using the
// NewCookieJar factory to create a suitable jar.
func NewHTTPClient(txp netxlite.HTTPTransport) *http.Client {
	return &http.Client{
		Transport: txp,
		Jar:       NewCookieJar(),
	}
}

// NewHTTPClientStdlib is a convenience factory for creating a new
// http.Client using NewHTTPClient and NewHTTPTransportStdlib.
func NewHTTPClientStdlib(logger netxlite.Logger, db DB) *http.Client {
	return NewHTTPClient(NewHTTPTransportStdlib(logger, db))
}

// NewHTTPRequestWithContext is a convenience factory for creating
// a new HTTP request with the typical headers we use when performing
// measurements already set inside of req.Header.
func NewHTTPRequestWithContext(ctx context.Context,
	method, URL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, URL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", httpheader.Accept())
	req.Header.Set("Accept-Language", httpheader.AcceptLanguage())
	req.Header.Set("User-Agent", httpheader.UserAgent())
	return req, nil
}

// NewHTTPGetRequest is a convenience factory for creating a new
// http.Request using the GET method and the given URL.
func NewHTTPGetRequest(ctx context.Context, URL string) (*http.Request, error) {
	return NewHTTPRequestWithContext(ctx, "GET", URL, nil)
}

// MustNewHTTPGetRequest is a convenience factory for creating
// a new http.Request using GET that panics on error.
func MustNewHTTPGetRequest(ctx context.Context, URL string) *http.Request {
	req, err := NewHTTPGetRequest(ctx, URL)
	runtimex.PanicOnError(err, "NewHTTPGetRequest failed")
	return req
}

package measuredb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// WCTHHTTPClient is the HTTP client type expected by
// the wtchWorker to query the test helper.
type WCTHHTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// wcthWorker is an implementation of the TestHelper interface
// that uses the Web Connectivity test helper.
type wcthWorker struct {
	db     DB
	logger netxlite.Logger
	clnt   WCTHHTTPClient
	URL    string
}

// NewWCTHWorker creates a new TestHelper instance using the
// web connectivity test helper protocol.
//
// Arguments
//
// - logger is the logger to use;
//
// - db is the database to use;
//
// - clnt is the HTTP client to use;
//
// - URL is the WCTH service URL.
//
// All arguments are mandatory.
//
// Run algorithm
//
// 1. recover URL of the current round trip from the database;
//
// 2. issue WCTH query with current URL, standard HTTP headers, and a
// list of TCP endpoints discovered by the probe;
//
// 3. format WCTH response to TestHelperMeasurement struct;
//
// 4. insert measurement into DB and return it.
//
// CAVEAT: this client can only work when the underlying database
// is using precise HTTP round trip measurements.
//
// CAVEAT: this implementation is very inefficient because the
// WCTH will fetch the whole redirection chain for every request
// but the WCTH is already there and it can bootstrap us.
func NewWCTHWorker(
	logger netxlite.Logger, db DB, clnt WCTHHTTPClient, URL string) TestHelper {
	return &wcthWorker{db: db, logger: logger, clnt: clnt, URL: URL}
}

var errWCTHRequestFailed = errors.New("wcth: request failed")

func (w *wcthWorker) Run(
	ctx context.Context, endpoints []string) (*TestHelperMeasurement, error) {
	URL, err := SelectURLFromHTTPRoundTripURLWithCurrentRoundTrip(w.db)
	if err != nil {
		return nil, err
	}
	wtchReq := &wcthRequest{
		HTTPRequest:        URL,
		HTTPRequestHeaders: NewHTTPRequestHeaderForMeasuring(),
		TCPConnect:         endpoints,
	}
	reqBody, err := json.Marshal(wtchReq)
	runtimex.PanicOnError(err, "json.Marshal failed")
	req, err := http.NewRequestWithContext(ctx, "POST", w.URL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("miniooni/%s", version.Version))
	resp, err := w.clnt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errWCTHRequestFailed
	}
	const maxBodySize = 1 << 20
	r := io.LimitReader(resp.Body, maxBodySize)
	respBody, err := iox.ReadAllContext(ctx, r)
	if err != nil {
		return nil, err
	}
	var wcthResp wcthResponse
	if err := json.Unmarshal(respBody, &wcthResp); err != nil {
		return nil, err
	}
	m := &TestHelperMeasurement{
		HTTPRoundTripID: w.db.HTTPRoundTripID(),
		DNSErr:          wcthResp.DNS.Failure,
		DNSAddrs:        wcthFilterDNSAddrs(wcthResp.DNS.Addrs),
		TCPEndpoints:    make(map[string]*string),
	}
	for addr, status := range wcthResp.TCPConnect {
		m.TCPEndpoints[addr] = status.Failure
	}
	w.db.InsertIntoTestHelperMeasurement(m)
	return m, nil
}

func wcthFilterDNSAddrs(addrs []string) (out []string) {
	for _, addr := range addrs {
		if net.ParseIP(addr) == nil {
			continue // WCTH also returns the CNAME
		}
		out = append(out, addr)
	}
	return
}

// wcthRequest is the request that we send to the control
type wcthRequest struct {
	HTTPRequest        string              `json:"http_request"`
	HTTPRequestHeaders map[string][]string `json:"http_request_headers"`
	TCPConnect         []string            `json:"tcp_connect"`
}

// wcthTCPConnectResult is the result of the TCP connect
// attempt performed by the control vantage point.
type wcthTCPConnectResult struct {
	Status  bool    `json:"status"`
	Failure *string `json:"failure"`
}

// wcthHTTPRequestResult is the result of the HTTP request
// performed by the control vantage point.
type wcthHTTPRequestResult struct {
	BodyLength int64             `json:"body_length"`
	Failure    *string           `json:"failure"`
	Title      string            `json:"title"`
	Headers    map[string]string `json:"headers"`
	StatusCode int64             `json:"status_code"`
}

// wcthDNSResult is the result of the DNS lookup
// performed by the control vantage point.
type wcthDNSResult struct {
	Failure *string  `json:"failure"`
	Addrs   []string `json:"addrs"`
	ASNs    []int64  `json:"-"` // not visible from the JSON
}

// wcthResponse is the response from the control service.
type wcthResponse struct {
	TCPConnect  map[string]wcthTCPConnectResult `json:"tcp_connect"`
	HTTPRequest wcthHTTPRequestResult           `json:"http_request"`
	DNS         wcthDNSResult                   `json:"dns"`
}

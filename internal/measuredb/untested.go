package measuredb

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// UntestedHTTPEndpointInstructions contains instructions
// for testing an endpoint that hasn't been tested. We
// build the instructions from the database.
type UntestedHTTPEndpointInstructions struct {
	Method   string      // method to use
	URL      *url.URL    // URL to use
	Header   http.Header // headers including cookies
	Protocol string      // protocol: one of "httpc", "https", "h3"
	SNI      string      // SNI
	ALPN     []string    // ALPN
	Domain   string      // domain
	Network  string      // network of the remote endpoint (e.g., "tcp")
	Address  string      // remote endpoint address
}

// NewUntestedHTTPEndpointInstructions builds instructions
// for testing untested endpoints based on the database. We return
// an error in case the input list is inconsistent with the
// current content of the database. On success, the output list
// may be empty if, e.g., the input list was empty.
//
// CAVEAT: this function will _only_ select the round trips
// in which we discovered endpoints. If a subsequent round
// trip uses a persistent network connection, its domain-endpoints
// table will be empty, and we will skip the round trip. We do
// not see this as a big issue because: (1) for HTTPS or QUIC we
// don't care; (2) for HTTP we mainly assume that censorship
// is implemented using _content_ based proxies, not endpoints.
func NewUntestedHTTPEndpointInstructions(
	db DB, epnts []*DomainEndpoint) ([]*UntestedHTTPEndpointInstructions, error) {
	var out []*UntestedHTTPEndpointInstructions
	for _, dep := range epnts {
		rtx, err := selectHTTPRoundTripWithRoundTripID(db, dep.RoundTripID)
		if err != nil {
			return nil, err
		}
		m := &UntestedHTTPEndpointInstructions{
			Method:  rtx.RequestMethod,
			URL:     rtx.RequestURL,
			Header:  rtx.RequestHeader,
			Domain:  dep.Domain,
			Network: dep.Network,
			Address: dep.Address,
		}
		thx, err := selectTLSHandshakeWithRoundTripID(db, dep.RoundTripID)
		if err != nil && m.Network == "tcp" && m.URL.Scheme == "https" {
			return nil, err
		}
		if thx != nil {
			m.Protocol = "https"
			m.SNI = thx.SNI
			m.ALPN = thx.ALPN
		} else if m.Network == "tcp" {
			m.Protocol = "httpc"
		}
		out = append(out, m)
	}
	return out, nil
}

// MeasureUntestedEndpoints measures the untested endpoints described
// by the input instructions sequentially. There is no result for
// this operation but the side effect of adding measurements to the DB.
//
// CAVEAT: even if we were attempting to run measurements in parallel
// the precise round trip measurement rule would prevent that.
func MeasureUntestedEndpoints(ctx context.Context, db DB,
	logger netxlite.Logger, instr ...*UntestedHTTPEndpointInstructions) {
	for _, uei := range instr {
		uei.Measure(ctx, db, logger)
	}
}

// Measure attempts the measure the endpoint described in the instructions
// using HTTP, HTTPS, or HTTP3. There is no result for this operation but
// the side effect of adding measurements to the DB.
//
// CAVEAT: this function assumes we are going to use the Go standard
// library to measure the untested endpoints. This fact may change in
// a more mature version of this implementation.
func (instr *UntestedHTTPEndpointInstructions) Measure(
	ctx context.Context, db DB, logger netxlite.Logger) {
	// QUIRK: here it suffices to use a connector because the
	// connect increments the endpoint ID. We should maybe zap
	// this comment when we have testing, but for now...
	cx := WrapConnector(db, netxlite.NewConnector(logger))
	conn, err := cx.DialContext(ctx, instr.Network, instr.Address)
	if err != nil {
		return
	}
	switch instr.Protocol {
	case "httpc":
		instr.httpc(ctx, db, logger, conn)
	case "https":
		instr.tls(ctx, db, logger, conn)
	default:
		// We don't have this case but closing the connection
		// and the connect measurement will be there
		conn.Close()
	}
}

func (instr *UntestedHTTPEndpointInstructions) tls(
	ctx context.Context, db DB, logger netxlite.Logger, conn net.Conn) {
	thx := WrapTLSHandshaker(db, netxlite.NewTLSHandshakerStdlib(logger))
	tconn, _, err := thx.Handshake(ctx, conn, &tls.Config{
		ServerName: instr.SNI,
		NextProtos: instr.ALPN,
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	if err != nil {
		conn.Close() // we own it
		return
	}
	instr.https(ctx, db, logger, tconn)
}

func (instr *UntestedHTTPEndpointInstructions) httpc(
	ctx context.Context, db DB, logger netxlite.Logger, conn net.Conn) {
	defer conn.Close() // we own it
	d := netxlite.NewSingleUseDialer(conn)
	td := netxlite.NewNullTLSDialer()
	instr.httpx(ctx, db, logger, d, td)
}

func (instr *UntestedHTTPEndpointInstructions) https(
	ctx context.Context, db DB, logger netxlite.Logger, conn net.Conn) {
	defer conn.Close() // we own it
	d := netxlite.NewNullDialer()
	// the following cast is safe because TLSHandshaker guarantees that
	td := netxlite.NewSingleUseTLSDialer(conn.(netxlite.TLSConn))
	instr.httpx(ctx, db, logger, d, td)
}

func (instr *UntestedHTTPEndpointInstructions) httpx(
	ctx context.Context, db DB, logger netxlite.Logger,
	d netxlite.Dialer, td netxlite.TLSDialer) {
	txp := netxlite.WrapHTTPTransport(logger, WrapHTTPTransport(
		db, netxlite.NewOOHTTPBaseTransport(d, td),
	))
	defer txp.CloseIdleConnections() // ensure we close idle conns
	req, err := http.NewRequestWithContext(
		ctx, instr.Method, instr.URL.String(), nil)
	if err != nil {
		return
	}
	req.Header = instr.Header
	resp, err := txp.RoundTrip(req)
	if err != nil {
		return
	}
	// Close the body to ensure we _only_ have idle conns. We do
	// not need to read it because this package's transport already
	// reads a small snapshot of every response body.
	resp.Body.Close()
}

package measuredb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// WrapTLSHandshaker wraps a TLSHandshaker to add measuredb capabilities.
func WrapTLSHandshaker(db DB, thx netxlite.TLSHandshaker) netxlite.TLSHandshaker {
	return &tlsHandshakerDB{TLSHandshaker: thx, DB: db}
}

type tlsHandshakerDB struct {
	netxlite.TLSHandshaker
	DB
}

// TLSHandshake contains a TLS handshake event.
//
// Note that EndpointID and RoundTripID only make sense when
// the DB we're using enforces precise HTTP round trips.
type TLSHandshake struct {
	EndpointID      int64     // Endpoint ID
	RoundTripID     int64     // HTTP round trip ID
	Engine          string    // engine we're using (e.g., "yawning")
	Network         string    // network (e.g., "tcp")
	RemoteAddr      string    // remote address (e.g., "1.1.1.1:443")
	LocalAddr       string    // local address
	SNI             string    // ServerName from tls.Config
	ALPN            []string  // NextProtos from tls.Config
	SkipVerify      bool      // InsecureSkipVerify from tls.Config
	Started         time.Time // when we started
	Finished        time.Time // when we finished
	Error           error     // error or nil
	TLSVersion      string    // TLSVersion from connection state
	CipherSuite     string    // cipher suite from connection state
	NegotiatedProto string    // negotiated protocol from connection state
	PeerCerts       [][]byte  // peer certs from connection state
}

func (thx *tlsHandshakerDB) Handshake(ctx context.Context,
	conn net.Conn, config *tls.Config) (net.Conn, tls.ConnectionState, error) {
	network := conn.RemoteAddr().Network()
	remoteAddr := conn.RemoteAddr().String()
	localAddr := conn.LocalAddr().String()
	started := time.Now()
	tconn, state, err := thx.TLSHandshaker.Handshake(ctx, conn, config)
	finished := time.Now()
	thx.DB.InsertIntoTLSHandshake(&TLSHandshake{
		EndpointID:      thx.DB.EndpointID(),
		RoundTripID:     thx.DB.HTTPRoundTripID(),
		Engine:          "", // TODO(bassosimone): add support
		Network:         network,
		RemoteAddr:      remoteAddr,
		LocalAddr:       localAddr,
		SNI:             config.ServerName,
		ALPN:            config.NextProtos,
		SkipVerify:      config.InsecureSkipVerify,
		Started:         started,
		Finished:        finished,
		Error:           err,
		TLSVersion:      netxlite.TLSVersionString(state.Version),
		CipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		NegotiatedProto: state.NegotiatedProtocol,
		PeerCerts:       peerCerts(err, &state),
	})
	return tconn, state, err
}

func peerCerts(err error, state *tls.ConnectionState) (out [][]byte) {
	var x509HostnameError x509.HostnameError
	if errors.As(err, &x509HostnameError) {
		// Test case: https://wrong.host.badssl.com/
		return [][]byte{x509HostnameError.Certificate.Raw}
	}
	var x509UnknownAuthorityError x509.UnknownAuthorityError
	if errors.As(err, &x509UnknownAuthorityError) {
		// Test case: https://self-signed.badssl.com/. This error has
		// never been among the ones returned by MK.
		return [][]byte{x509UnknownAuthorityError.Cert.Raw}
	}
	var x509CertificateInvalidError x509.CertificateInvalidError
	if errors.As(err, &x509CertificateInvalidError) {
		// Test case: https://expired.badssl.com/
		return [][]byte{x509CertificateInvalidError.Cert.Raw}
	}
	for _, cert := range state.PeerCertificates {
		out = append(out, cert.Raw)
	}
	return
}

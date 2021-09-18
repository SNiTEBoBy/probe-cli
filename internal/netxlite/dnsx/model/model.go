// Package model contains the dnsx model.
package model

// HTTPS is an HTTPS reply.
type HTTPS interface {
	// ALPN returns the ALPNs inside the SVCBAlpn structure
	ALPN() []string
}

package measuredb

// This file contains the generic spec of a test helper. We use
// a test helper during the dial phase to gather more information
// on the endpoints that we're about to test.

import (
	"context"
	"errors"
)

// TestHelperMeasurement contains the result of the
// measurement performed by the test helper.
//
// The HTTPRoundTripID is only meaningful when the underlying
// database supports precise HTTP round trip measurements.
type TestHelperMeasurement struct {
	// HTTPRoundTripID is the current HTTP round trip ID.
	HTTPRoundTripID int64

	// DNSErr is the DNS error seen by the test helper. This field
	// is nil on success and contains an error string otherwise.
	DNSErr *string

	// DNSAddrs contains the addresses discovered by the test
	// helper using its own resolver.
	DNSAddrs []string

	// TCPEndpoints contains the TCP endpoints tested by the test
	// helper. We map each endpoint to its optional error.
	TCPEndpoints map[string]*string
}

// TestHelper is the generic interface of a test helper. We currently
// use the Web Connectivity test helper by default. We will migrate to
// a more flexible test helper in the future. We use an interface for
// testing and because it allows for smooth upgrading from a given test
// helper implementation to another one.
type TestHelper interface {
	// Run runs the test helper query and returns the result
	// of running such a query or an error.
	Run(ctx context.Context, endpoints []string) (*TestHelperMeasurement, error)
}

// ErrNoConfiguredTestHelper indicates you didn't configure a TestHelper.
var ErrNoConfiguredTestHelper = errors.New("no configured TestHelper")

// NullTestHelper is a test helper that returns ErrNoConfiguredTestHelper.
type NullTestHelper struct{}

var _ TestHelper = &NullTestHelper{}

func (th *NullTestHelper) Run(
	ctx context.Context, endpoints []string) (*TestHelperMeasurement, error) {
	return nil, ErrNoConfiguredTestHelper
}

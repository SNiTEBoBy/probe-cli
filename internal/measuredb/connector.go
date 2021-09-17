package measuredb

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// WrapConnector wraps a Connector to add measuredb capabilities.
func WrapConnector(db DB, c netxlite.Connector) netxlite.Connector {
	return &connectorDB{Connector: c, DB: db}

}

type connectorDB struct {
	netxlite.Connector
	DB
}

// Connection contains a connection event.
//
// EndpointID and RoundTripID only make sense when we are
// using precise HTTP round trip measurements.
//
// On the contrary ConnID is always valid. It only becomes
// useful, though, w/o precise round trip measurements.
type Connection struct {
	EndpointID  int64     // endpoint ID
	RoundTripID int64     // HTTP round trip ID
	Operation   string    // operation name
	ConnID      int64     // connection ID
	Network     string    // network ("tcp" or "udp")
	RemoteAddr  string    // remote address (e.g., "1.1.1.1:443")
	LocalAddr   string    // local address
	Started     time.Time // operation start
	Finished    time.Time // operation end
	Error       error     // error or nil
	Count       int       // #bytes for I/O operations
}

func (c *connectorDB) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	c.DB.OnTryEndpoint(network, address) // bumps the EndpointID
	connID := nextConnID()               // bumps the ConnID
	started := time.Now()
	conn, err := c.Connector.DialContext(ctx, network, address)
	finished := time.Now()
	c.DB.InsertIntoConnection(&Connection{
		EndpointID:  c.DB.EndpointID(),
		RoundTripID: c.DB.HTTPRoundTripID(),
		Operation:   "connect",
		ConnID:      connID,
		Network:     network,
		RemoteAddr:  address,
		LocalAddr:   c.localAddrIfNotNil(conn),
		Started:     started,
		Finished:    finished,
		Error:       err,
		Count:       0,
	})
	if conn != nil {
		conn = &connDB{
			Conn:       conn,
			DB:         c.DB,
			endpointID: c.DB.EndpointID(),
			connID:     connID,
			remoteAddr: address,
			localAddr:  conn.LocalAddr().String(),
			network:    network,
		}
	}
	return conn, err
}

func (c *connectorDB) localAddrIfNotNil(conn net.Conn) (addr string) {
	if conn != nil {
		addr = conn.LocalAddr().String()
	}
	return
}

type connDB struct {
	net.Conn
	DB
	endpointID int64
	connID     int64
	remoteAddr string
	localAddr  string
	network    string
}

func (c *connDB) Read(b []byte) (int, error) {
	started := time.Now()
	count, err := c.Conn.Read(b)
	finished := time.Now()
	c.DB.InsertIntoConnection(&Connection{
		EndpointID:  c.endpointID,
		RoundTripID: c.DB.HTTPRoundTripID(),
		Operation:   "read",
		ConnID:      c.connID,
		Network:     c.network,
		RemoteAddr:  c.remoteAddr,
		LocalAddr:   c.localAddr,
		Started:     started,
		Finished:    finished,
		Error:       err,
		Count:       count,
	})
	return count, err
}

func (c *connDB) Write(b []byte) (int, error) {
	started := time.Now()
	count, err := c.Conn.Write(b)
	finished := time.Now()
	c.DB.InsertIntoConnection(&Connection{
		EndpointID:  c.endpointID,
		RoundTripID: c.DB.HTTPRoundTripID(),
		Operation:   "write",
		ConnID:      c.connID,
		Network:     c.network,
		RemoteAddr:  c.remoteAddr,
		LocalAddr:   c.localAddr,
		Started:     started,
		Finished:    finished,
		Error:       err,
		Count:       count,
	})
	return count, err
}

func (c *connDB) Close() error {
	started := time.Now()
	err := c.Conn.Close()
	finished := time.Now()
	c.DB.InsertIntoConnection(&Connection{
		EndpointID:  c.endpointID,
		RoundTripID: c.DB.HTTPRoundTripID(),
		Operation:   "close",
		ConnID:      c.connID,
		Network:     c.network,
		RemoteAddr:  c.remoteAddr,
		LocalAddr:   c.localAddr,
		Started:     started,
		Finished:    finished,
		Error:       err,
		Count:       0,
	})
	return err
}

var connID = &atomicx.Int64{}

func nextConnID() int64 {
	return connID.Add(1)
}

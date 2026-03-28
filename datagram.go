package wt

import "fmt"

// MaxDatagramSize is the recommended maximum datagram payload size.
// QUIC datagrams are limited by the path MTU minus QUIC overhead.
// Typical safe size is ~1200 bytes (minimum QUIC MTU of 1280 minus headers).
// Larger datagrams may be fragmented or dropped.
const MaxDatagramSize = 1200

// ValidateDatagramSize checks if a datagram payload is within safe limits.
// Returns an error if the payload is too large.
func ValidateDatagramSize(data []byte) error {
	if len(data) > MaxDatagramSize {
		return fmt.Errorf("wt: datagram payload %d bytes exceeds safe limit of %d bytes",
			len(data), MaxDatagramSize)
	}
	return nil
}

// SendDatagramSafe sends a datagram with size validation.
// Returns an error if the payload exceeds MaxDatagramSize.
func (c *Context) SendDatagramSafe(data []byte) error {
	if err := ValidateDatagramSize(data); err != nil {
		return err
	}
	return c.SendDatagram(data)
}

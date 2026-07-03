// Package comm defines the communication protocol, packet encoding, and
// connection hub shared by all transport layers (WebSocket, UDP, TCP).
//
// The binary packet format is transport-agnostic — the same bytes flow over
// WebSocket frames, TCP streams, or UDP datagrams. See udp-tcp-protocol.md
// for the full specification.
package comm

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ── Protocol constants ──────────────────────────────────────────────────

// ProtocolVersion is the current wire-format version. Increment when the
// header layout changes in a backward-incompatible way.
const ProtocolVersion uint8 = 1

// Packet types.
const (
	TypeACK    uint8 = 0 // acknowledgement of receipt
	TypeMedia  uint8 = 1 // audio / video frame
	TypeMSG    uint8 = 2 // chat message or signalling (SDP, ICE)
	TypeJoin   uint8 = 3 // peer connected (server → clients)
	TypeLeave  uint8 = 4 // peer disconnected (server → clients)
	TypeCanvas uint8 = 5 // live canvas draw / stroke data
	TypePing   uint8 = 6 // latency probe (server → client)
	TypePong   uint8 = 7 // latency reply  (client → server)
)

// HeaderSize is the fixed byte-length of every packet header.
//
//	Byte 0    Version
//	Byte 1    Type
//	Bytes 2–3 Payload Length (uint16, big-endian)
const HeaderSize = 4

// MaxPayloadSize is the largest payload the header can encode (64 KiB).
const MaxPayloadSize = 65535

// ── Packet ──────────────────────────────────────────────────────────────

// Packet is a single protocol message. It is the unit of exchange between
// peers and the server.
type Packet struct {
	Version uint8
	Type    uint8
	Payload []byte
}

// NewPacket builds a Packet with the current protocol version and the given
// type + payload. It is the caller's responsibility to keep the payload
// under MaxPayloadSize.
func NewPacket(t uint8, payload []byte) Packet {
	return Packet{Version: ProtocolVersion, Type: t, Payload: payload}
}

// Encode writes the packet's binary representation to w.
func (p Packet) Encode(w io.Writer) error {
	header := make([]byte, HeaderSize)
	header[0] = p.Version
	header[1] = p.Type
	binary.BigEndian.PutUint16(header[2:4], uint16(len(p.Payload)))

	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("comm: write header: %w", err)
	}
	if len(p.Payload) > 0 {
		if _, err := w.Write(p.Payload); err != nil {
			return fmt.Errorf("comm: write payload: %w", err)
		}
	}
	return nil
}

// EncodeBytes returns the packet encoded as a byte slice. Useful when the
// caller needs the raw bytes (e.g. for sending over UDP).
func (p Packet) EncodeBytes() []byte {
	buf := make([]byte, HeaderSize+len(p.Payload))
	buf[0] = p.Version
	buf[1] = p.Type
	binary.BigEndian.PutUint16(buf[2:4], uint16(len(p.Payload)))
	copy(buf[HeaderSize:], p.Payload)
	return buf
}

// DecodePacket reads one packet from r. Returns an error if the version is
// unknown or the header/payload cannot be read in full.
func DecodePacket(r io.Reader) (Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return Packet{}, fmt.Errorf("comm: read header: %w", err)
	}

	version := header[0]
	if version != ProtocolVersion {
		return Packet{}, fmt.Errorf("comm: unknown protocol version %d (expected %d)", version, ProtocolVersion)
	}

	ptype := header[1]
	length := binary.BigEndian.Uint16(header[2:4])

	var payload []byte
	if length > 0 {
		payload = make([]byte, length)
		if _, err := io.ReadFull(r, payload); err != nil {
			return Packet{}, fmt.Errorf("comm: read payload: %w", err)
		}
	}

	return Packet{Version: version, Type: ptype, Payload: payload}, nil
}

// TypeString returns a human-readable name for the packet type.
func TypeString(t uint8) string {
	switch t {
	case TypeACK:
		return "ACK"
	case TypeMedia:
		return "Media"
	case TypeMSG:
		return "MSG"
	case TypeJoin:
		return "Join"
	case TypeLeave:
		return "Leave"
	case TypeCanvas:
		return "Canvas"
	case TypePing:
		return "Ping"
	case TypePong:
		return "Pong"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

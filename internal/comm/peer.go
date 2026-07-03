package comm

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const writeTimeout = 10 * time.Second

// ── Peer ────────────────────────────────────────────────────────────────

type Peer struct {
	ID       string
	Username string
	avatar   string
	latency  int64  // last measured RTT in ms

	conn *websocket.Conn
	send chan Packet

	closeOnce sync.Once
}

func NewPeer(id string, conn *websocket.Conn) *Peer {
	return &Peer{
		ID:   id,
		conn: conn,
		send: make(chan Packet, 64),
	}
}

func (p *Peer) PID() string                { return p.ID }
func (p *Peer) AvatarURL() string          { return p.avatar }
func (p *Peer) SetAvatarURL(url string)    { p.avatar = url }
func (p *Peer) SetDisplayName(name string) { p.Username = name }
func (p *Peer) Latency() int64              { return p.latency }
func (p *Peer) SetLatency(ms int64)         { p.latency = ms }

func (p *Peer) DisplayName() string {
	if p.Username != "" {
		return p.Username
	}
	if len(p.ID) > 8 {
		return "WS-" + p.ID[:8]
	}
	return "WS-" + p.ID
}

func (p *Peer) Send(packet Packet) error {
	select {
	case p.send <- packet:
		return nil
	default:
		return nil
	}
}

func (p *Peer) Close() {
	p.closeOnce.Do(func() {
		close(p.send)
		p.conn.Close()
	})
}

// ── Read pump ───────────────────────────────────────────────────────────

func (p *Peer) ReadPump(hub *Hub) {
	defer func() {
		hub.unregister <- p
		p.Close()
	}()

	for {
		_, r, err := p.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				LogError("WS read error from %s: %v", shortPID(p.ID), err)
			}
			return
		}

		packet, err := DecodePacket(r)
		if err != nil {
			LogError("WS decode error from %s: %v", shortPID(p.ID), err)
			return
		}

		hub.Push(p, packet)
	}
}

// ── Write pump ──────────────────────────────────────────────────────────

func (p *Peer) WritePump() {
	defer p.Close()

	for packet := range p.send {
		p.conn.SetWriteDeadline(time.Now().Add(writeTimeout))

		w, err := p.conn.NextWriter(websocket.BinaryMessage)
		if err != nil {
			LogError("WS write error to %s: %v", shortPID(p.ID), err)
			return
		}

		if err := packet.Encode(w); err != nil {
			LogError("WS encode error to %s: %v", shortPID(p.ID), err)
			w.Close()
			return
		}

		if err := w.Close(); err != nil {
			LogError("WS flush error to %s: %v", shortPID(p.ID), err)
			return
		}
	}
}

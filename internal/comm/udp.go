package comm

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// ── UDPPeer ─────────────────────────────────────────────────────────────

type UDPPeer struct {
	id       string
	username string
	avatar   string
	latency  int64
	lastSeen time.Time
	addr     *net.UDPAddr
	conn     *net.UDPConn
	onClose  func()

	mu        sync.RWMutex
	closeOnce sync.Once
}

func (u *UDPPeer) PID() string { return u.id }

func (u *UDPPeer) DisplayName() string {
	u.mu.RLock()
	username := u.username
	id := u.id
	u.mu.RUnlock()

	if username != "" {
		return username
	}
	if len(id) > 8 {
		return "UDP-" + id[:8]
	}
	return "UDP-" + id
}

func (u *UDPPeer) SetDisplayName(name string) {
	u.mu.Lock()
	u.username = name
	u.mu.Unlock()
}

func (u *UDPPeer) AvatarURL() string {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.avatar
}

func (u *UDPPeer) SetAvatarURL(url string) {
	u.mu.Lock()
	u.avatar = url
	u.mu.Unlock()
}

func (u *UDPPeer) Latency() int64 {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.latency
}

func (u *UDPPeer) SetLatency(ms int64) {
	u.mu.Lock()
	u.latency = ms
	u.lastSeen = time.Now()
	u.mu.Unlock()
}

func (u *UDPPeer) MarkSeen() {
	u.mu.Lock()
	u.lastSeen = time.Now()
	u.mu.Unlock()
}

func (u *UDPPeer) IsStale(now time.Time, timeout time.Duration) bool {
	u.mu.RLock()
	lastSeen := u.lastSeen
	u.mu.RUnlock()
	return !lastSeen.IsZero() && now.Sub(lastSeen) > timeout
}

func (u *UDPPeer) Send(packet Packet) error {
	data := packet.EncodeBytes()
	_, err := u.conn.WriteToUDP(data, u.addr)
	if err != nil {
		LogError("UDP write to %s (%s): %v", shortPID(u.id), u.addr, err)
	}
	return err
}

func (u *UDPPeer) Close() {
	u.closeOnce.Do(func() {
		if u.onClose != nil {
			u.onClose()
		}
	})
}

// ── UDP listener ────────────────────────────────────────────────────────

func ListenUDP(addr string, hub *Hub) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("comm: resolve UDP %s: %w", addr, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("comm: bind UDP %s: %w", addr, err)
	}
	defer conn.Close()

	LogSystem("UDP listener ready on %s", addr)

	addrMap := make(map[string]*UDPPeer)
	var addrMu sync.Mutex
	buf := make([]byte, MaxPayloadSize+HeaderSize)

	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			LogError("UDP read: %v", err)
			continue
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		packet, err := decodeFromBytes(data)
		if err != nil {
			LogError("UDP decode from %s: %v — %d raw bytes", remote, err, n)
			continue
		}

		key := remote.String()

		isNew := false
		addrMu.Lock()
		peer, ok := addrMap[key]
		if !ok {
			if !shouldRegisterUDPPacket(packet.Type) {
				addrMu.Unlock()
				LogWarn("ignoring UDP %s from unknown %s", typeLabel(packet.Type), remote)
				continue
			}
			pid := NewPeerID()
			peer = &UDPPeer{
				id:       pid,
				addr:     remote,
				conn:     conn,
				lastSeen: time.Now(),
				onClose: func() {
					addrMu.Lock()
					delete(addrMap, key)
					addrMu.Unlock()
				},
			}
			addrMap[key] = peer
			isNew = true
		}
		peer.MarkSeen()
		addrMu.Unlock()

		if isNew {
			hub.Register(peer)
		}
		hub.Push(peer, packet)
	}
}

func shouldRegisterUDPPacket(packetType uint8) bool {
	switch packetType {
	case TypeMSG, TypeMedia, TypeCanvas:
		return true
	default:
		return false
	}
}

// ── Decode from raw bytes ──────────────────────────────────────────────

func decodeFromBytes(data []byte) (Packet, error) {
	if len(data) < HeaderSize {
		return Packet{}, fmt.Errorf("datagram too short: %d bytes", len(data))
	}

	version := data[0]
	if version != ProtocolVersion {
		return Packet{}, fmt.Errorf("unknown version %d", version)
	}

	ptype := data[1]
	length := binary.BigEndian.Uint16(data[2:4])

	if int(HeaderSize+length) > len(data) {
		return Packet{}, fmt.Errorf("payload len %d exceeds datagram size %d", length, len(data)-HeaderSize)
	}

	payload := make([]byte, length)
	copy(payload, data[HeaderSize:HeaderSize+length])

	return Packet{Version: version, Type: ptype, Payload: payload}, nil
}

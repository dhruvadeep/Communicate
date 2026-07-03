package comm

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

// ── inbound message ─────────────────────────────────────────────────────

type inboundMsg struct {
	peer   PeerConn
	packet Packet
}

// ── Hub ─────────────────────────────────────────────────────────────────

type AuthFunc func(token string) (username string, avatarURL string, ok bool)

type Hub struct {
	peers map[string]PeerConn

	register   chan PeerConn
	unregister chan PeerConn
	inbound    chan inboundMsg

	OnAuth AuthFunc
}

func NewHub() *Hub {
	return &Hub{
		peers:      make(map[string]PeerConn),
		register:   make(chan PeerConn),
		unregister: make(chan PeerConn),
		inbound:    make(chan inboundMsg, 256),
	}
}

func (h *Hub) Run() {
	// Ping every peer every 10 s to measure latency.
	pingTick := time.NewTicker(10 * time.Second)
	defer pingTick.Stop()

	for {
		select {
		case peer := <-h.register:
			h.onRegister(peer)

		case peer := <-h.unregister:
			h.onUnregister(peer)

		case msg := <-h.inbound:
			h.route(msg)

		case <-pingTick.C:
			h.pingAll()
		}
	}
}

// ── Registration ────────────────────────────────────────────────────────

func (h *Hub) onRegister(peer PeerConn) {
	welcome, _ := json.Marshal(joinPayload{PeerID: peer.PID()})
	welcomePkt := NewPacket(TypeJoin, welcome)
	peer.Send(welcomePkt)
	LogTx(peer, welcomePkt)

	h.peers[peer.PID()] = peer

	transport := "ws"
	if _, ok := peer.(*UDPPeer); ok {
		transport = "udp"
	}
	LogRegister(peer, len(h.peers), transport)

	h.broadcastPeerList()
}

func (h *Hub) onUnregister(peer PeerConn) {
	pid := peer.PID()
	if _, ok := h.peers[pid]; !ok {
		return
	}

	delete(h.peers, pid)
	peer.Close()

	LogUnregister(peer, len(h.peers))

	h.broadcastPeerList()
}

func (h *Hub) broadcastPeerList() {
	list := make([]PeerInfo, 0, len(h.peers))
	for _, p := range h.peers {
		list = append(list, PeerInfo{
			PeerID:    p.PID(),
			Username:  p.DisplayName(),
			Avatar:    p.AvatarURL(),
			LatencyMs: p.Latency(),
		})
	}

	payload, _ := json.Marshal(map[string]interface{}{
		"event": "peer_list",
		"peers": list,
	})

	pkt := NewPacket(TypeJoin, payload)
	for _, p := range h.peers {
		p.Send(pkt)
	}

	LogTxBroadcast(pkt, len(h.peers))
}

// ── Ping / latency ──────────────────────────────────────────────────────

var pingSeq int64

const udpPeerTimeout = 35 * time.Second

func (h *Hub) pingAll() {
	if len(h.peers) == 0 {
		return
	}
	pingSeq++
	now := time.Now()
	payload, _ := json.Marshal(map[string]interface{}{
		"seq": pingSeq,
		"ts":  now.UnixMilli(),
	})
	pkt := NewPacket(TypePing, payload)
	for _, p := range h.peers {
		if udpPeer, ok := p.(*UDPPeer); ok && udpPeer.IsStale(now, udpPeerTimeout) {
			LogWarn("UDP peer %s timed out after %s without traffic",
				shortPID(p.PID()), udpPeerTimeout)
			h.onUnregister(p)
			continue
		}
		p.Send(pkt)
	}
}

func (h *Hub) handlePong(msg inboundMsg) {
	if _, ok := h.peers[msg.peer.PID()]; !ok {
		return
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(msg.packet.Payload, &envelope); err != nil {
		return
	}
	ts, _ := envelope["ts"].(float64)
	if ts > 0 {
		rtt := time.Now().UnixMilli() - int64(ts)
		msg.peer.SetLatency(rtt)
		LogPing(msg.peer, rtt)
		h.broadcastPeerList()
	}
}

// ── Routing ─────────────────────────────────────────────────────────────

func (h *Hub) route(msg inboundMsg) {
	if _, ok := h.peers[msg.peer.PID()]; !ok {
		return
	}

	LogRx(msg.peer, msg.packet)

	switch msg.packet.Type {

	case TypeMSG:
		h.routeMSG(msg)

	case TypeMedia:
		h.routeMedia(msg)

	case TypeCanvas:
		h.routeCanvas(msg)

	case TypePong:
		h.handlePong(msg)

	case TypeACK:
		// Consumed.

	case TypePing:
		// Client pinged us — echo back as PONG so it can measure RTT.
		pongPkt := NewPacket(TypePong, msg.packet.Payload)
		msg.peer.Send(pongPkt)

	case TypeLeave:
		h.onUnregister(msg.peer)

	case TypeJoin:
		LogWarn("client sent server-only type %s — ignored", typeLabel(msg.packet.Type))

	default:
		LogWarn("unknown packet type %d from %s — ignored",
			msg.packet.Type, shortPID(msg.peer.PID()))
	}
}

// ── MSG routing ────────────────────────────────────────────────────────

func (h *Hub) routeMSG(msg inboundMsg) {
	var envelope map[string]interface{}
	if err := json.Unmarshal(msg.packet.Payload, &envelope); err != nil {
		LogError("bad JSON from %s: %v", shortPID(msg.peer.PID()), err)
		errJSON, _ := json.Marshal(map[string]string{"error": "invalid JSON payload"})
		errPkt := NewPacket(TypeMSG, errJSON)
		msg.peer.Send(errPkt)
		LogTx(msg.peer, errPkt)
		return
	}

	// Auth token (UDP mid-session).
	if token, ok := envelope["token"].(string); ok && token != "" {
		if h.OnAuth != nil {
			username, avatar, ok := h.OnAuth(token)
			if ok {
				msg.peer.SetDisplayName(username)
				if avatar != "" {
					msg.peer.SetAvatarURL(avatar)
				}
				LogSystem("UDP peer %s authenticated as %s", shortPID(msg.peer.PID()), username)
				h.broadcastPeerList()
			} else {
				LogWarn("UDP auth failed for %s — invalid token", shortPID(msg.peer.PID()))
			}
		}
		return
	}

	envelope["from"] = msg.peer.PID()

	payload, err := json.Marshal(envelope)
	if err != nil {
		return
	}

	targetID, _ := envelope["to"].(string)

	if targetID != "" {
		if target, ok := h.peers[targetID]; ok {
			out := NewPacket(TypeMSG, payload)
			target.Send(out)
			LogTx(target, out)
		} else {
			LogWarn("target %s not found (from %s)", shortPID(targetID), shortPID(msg.peer.PID()))
			errJSON, _ := json.Marshal(map[string]string{"error": "peer not found", "peer": targetID})
			errPkt := NewPacket(TypeMSG, errJSON)
			msg.peer.Send(errPkt)
			LogTx(msg.peer, errPkt)
		}
		return
	}

	out := NewPacket(TypeMSG, payload)
	count := 0
	for id, p := range h.peers {
		if id != msg.peer.PID() {
			p.Send(out)
			count++
		}
	}
	LogTxBroadcast(out, count)
}

// ── Canvas / Media ──────────────────────────────────────────────────────

func (h *Hub) routeCanvas(msg inboundMsg) {
	count := 0
	for id, p := range h.peers {
		if id != msg.peer.PID() {
			p.Send(msg.packet)
			count++
		}
	}
	LogTxBroadcast(msg.packet, count)
}

func (h *Hub) routeMedia(msg inboundMsg) {
	count := 0
	for id, p := range h.peers {
		if id != msg.peer.PID() {
			p.Send(msg.packet)
			count++
		}
	}
	LogTxBroadcast(msg.packet, count)
}

// ── Helpers ─────────────────────────────────────────────────────────────

func (h *Hub) Register(p PeerConn)   { h.register <- p }
func (h *Hub) Unregister(p PeerConn) { h.unregister <- p }
func (h *Hub) Push(peer PeerConn, packet Packet) {
	h.inbound <- inboundMsg{peer: peer, packet: packet}
}

func NewPeerID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("comm: crypto/rand.Read failed: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}

type joinPayload struct {
	PeerID string `json:"peer_id,omitempty"`
	Event  string `json:"event,omitempty"`
}

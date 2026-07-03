package comm

import (
	"encoding/json"
	"testing"
	"time"
)

type hubTestPeer struct {
	id       string
	username string
	avatar   string
	latency  int64
	sent     []Packet
	closed   bool
}

func (p *hubTestPeer) PID() string { return p.id }

func (p *hubTestPeer) DisplayName() string {
	if p.username != "" {
		return p.username
	}
	return p.id
}

func (p *hubTestPeer) SetDisplayName(name string) { p.username = name }
func (p *hubTestPeer) AvatarURL() string          { return p.avatar }
func (p *hubTestPeer) SetAvatarURL(url string)    { p.avatar = url }
func (p *hubTestPeer) Latency() int64             { return p.latency }
func (p *hubTestPeer) SetLatency(ms int64)        { p.latency = ms }

func (p *hubTestPeer) Send(packet Packet) error {
	p.sent = append(p.sent, packet)
	return nil
}

func (p *hubTestPeer) Close() { p.closed = true }

func TestUDPPeerStoresLatencyAndAvatar(t *testing.T) {
	peer := &UDPPeer{id: "udp-peer"}

	peer.SetLatency(42)
	peer.SetAvatarURL("https://example.test/avatar.png")

	if got := peer.Latency(); got != 42 {
		t.Fatalf("Latency() = %d, want 42", got)
	}
	if got := peer.AvatarURL(); got != "https://example.test/avatar.png" {
		t.Fatalf("AvatarURL() = %q, want stored avatar URL", got)
	}
}

func TestHandlePongBroadcastsPeerListWithLatency(t *testing.T) {
	hub := NewHub()
	udpPeer := &hubTestPeer{id: "udp-a", username: "UDP-a"}
	wsPeer := &hubTestPeer{id: "ws-b", username: "WS-b"}
	hub.peers[udpPeer.id] = udpPeer
	hub.peers[wsPeer.id] = wsPeer

	payload, err := json.Marshal(map[string]int64{
		"seq": 1,
		"ts":  time.Now().Add(-25 * time.Millisecond).UnixMilli(),
	})
	if err != nil {
		t.Fatal(err)
	}

	hub.route(inboundMsg{
		peer:   udpPeer,
		packet: NewPacket(TypePong, payload),
	})

	if udpPeer.latency <= 0 {
		t.Fatalf("latency = %d, want a measured RTT", udpPeer.latency)
	}

	list := lastPeerList(t, wsPeer)
	info := findPeerInfo(list, udpPeer.id)
	if info == nil {
		t.Fatalf("peer_list missing %s: %#v", udpPeer.id, list)
	}
	if info.LatencyMs <= 0 {
		t.Fatalf("peer_list latency_ms = %d, want measured RTT", info.LatencyMs)
	}
}

func TestRouteLeaveUnregistersAndBroadcastsPeerList(t *testing.T) {
	hub := NewHub()
	leaving := &hubTestPeer{id: "udp-a", username: "UDP-a"}
	remaining := &hubTestPeer{id: "ws-b", username: "WS-b"}
	hub.peers[leaving.id] = leaving
	hub.peers[remaining.id] = remaining

	hub.route(inboundMsg{
		peer:   leaving,
		packet: NewPacket(TypeLeave, nil),
	})

	if _, ok := hub.peers[leaving.id]; ok {
		t.Fatalf("leaving peer still registered")
	}
	if !leaving.closed {
		t.Fatalf("leaving peer was not closed")
	}

	list := lastPeerList(t, remaining)
	if len(list) != 1 || list[0].PeerID != remaining.id {
		t.Fatalf("peer_list after leave = %#v, want only remaining peer", list)
	}
}

func TestShouldRegisterUDPPacketRejectsControlPackets(t *testing.T) {
	for _, packetType := range []uint8{TypeACK, TypeJoin, TypeLeave, TypePing, TypePong} {
		if shouldRegisterUDPPacket(packetType) {
			t.Fatalf("shouldRegisterUDPPacket(%s) = true, want false", TypeString(packetType))
		}
	}

	for _, packetType := range []uint8{TypeMSG, TypeMedia, TypeCanvas} {
		if !shouldRegisterUDPPacket(packetType) {
			t.Fatalf("shouldRegisterUDPPacket(%s) = false, want true", TypeString(packetType))
		}
	}
}

func lastPeerList(t *testing.T, peer *hubTestPeer) []PeerInfo {
	t.Helper()

	for i := len(peer.sent) - 1; i >= 0; i-- {
		packet := peer.sent[i]
		if packet.Type != TypeJoin {
			continue
		}

		var payload struct {
			Event string     `json:"event"`
			Peers []PeerInfo `json:"peers"`
		}
		if err := json.Unmarshal(packet.Payload, &payload); err != nil {
			t.Fatalf("decode peer_list: %v", err)
		}
		if payload.Event == "peer_list" {
			return payload.Peers
		}
	}

	t.Fatalf("no peer_list packet sent to %s", peer.id)
	return nil
}

func findPeerInfo(list []PeerInfo, peerID string) *PeerInfo {
	for i := range list {
		if list[i].PeerID == peerID {
			return &list[i]
		}
	}
	return nil
}

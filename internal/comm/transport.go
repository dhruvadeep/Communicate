package comm

// PeerConn is the minimal interface a transport must expose so the Hub can
// register, route to, and disconnect it. Both WebSocket peers and UDP peers
// implement this.
type PeerConn interface {
	PID() string

	DisplayName() string
	SetDisplayName(name string)

	AvatarURL() string
	SetAvatarURL(url string)

	// Latency returns the last measured round-trip time in milliseconds.
	// Returns 0 if not yet measured.
	Latency() int64
	SetLatency(ms int64)

	Send(Packet) error
	Close()
}

// PeerInfo is the public summary broadcast in peer-list updates.
type PeerInfo struct {
	PeerID    string `json:"peer_id"`
	Username  string `json:"username"`
	Avatar    string `json:"avatar,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}

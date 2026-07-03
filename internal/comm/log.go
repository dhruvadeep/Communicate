package comm

import (
	"fmt"
	"log"
	"time"
)

// ── ANSI colours ───────────────────────────────────────────────────────

const (
	cReset   = "\033[0m"
	cBold    = "\033[1m"
	cDim     = "\033[2m"
	cRed     = "\033[31m"
	cGreen   = "\033[32m"
	cYellow  = "\033[33m"
	cBlue    = "\033[34m"
	cMagenta = "\033[35m"
	cCyan    = "\033[36m"
)

// ── Public logging functions ───────────────────────────────────────────
//
// Every log line has the form:
//
//	HH:MM:SS  symbol TYPE  info…
//	                  └  detail (on next line, indented)

// LogRx logs a packet RECEIVED from a peer.
//
//	↓ MSG    47B  from b2641982 (malakar)
//	          └  {"text":"Hi"}
func LogRx(peer PeerConn, pkt Packet) {
	ts := time.Now().Format("15:04:05")
	name := peerLabel(peer)
	preview := trimPayload(pkt.Payload)
	log.Printf("%s%s ↓ %s%-6s %s%4dB%s  from %s%s%s%s",
		ts, cGreen, cBold, typeLabel(pkt.Type), cReset, len(pkt.Payload)+HeaderSize, cGreen, cReset, shortPID(peer.PID()), name, cReset)
	log.Printf("           %s└%s  %s", cDim, cReset, preview)
}

// LogTx logs a packet SENT to a specific peer.
//
//	↑ MSG    59B  to   6b2cab87 (142201026)
//	          └  {"from":"…","text":"Hi"}
func LogTx(peer PeerConn, pkt Packet) {
	ts := time.Now().Format("15:04:05")
	name := peerLabel(peer)
	preview := trimPayload(pkt.Payload)
	log.Printf("%s%s ↑ %s%-6s %s%4dB%s  to   %s%s%s%s",
		ts, cBlue, cBold, typeLabel(pkt.Type), cReset, len(pkt.Payload)+HeaderSize, cBlue, cReset, shortPID(peer.PID()), name, cReset)
	log.Printf("           %s└%s  %s", cDim, cReset, preview)
}

// LogTxBroadcast logs a packet SENT to all peers.
//
//	↑ MSG    59B  → broadcast (2 peers)
//	          └  {"text":"hello all"}
func LogTxBroadcast(pkt Packet, count int) {
	ts := time.Now().Format("15:04:05")
	preview := trimPayload(pkt.Payload)
	log.Printf("%s%s ↑ %s%-6s %s%4dB%s  %s→ broadcast (%d peers)%s",
		ts, cBlue, cBold, typeLabel(pkt.Type), cReset, len(pkt.Payload)+HeaderSize, cBlue, cDim, count, cReset)
	log.Printf("           %s└%s  %s", cDim, cReset, preview)
}

// LogRegister logs a peer connecting.
//
//	◆ JOIN   b2641982 (malakar)  [udp]  3 peers now
func LogRegister(peer PeerConn, peerCount int, transport string) {
	ts := time.Now().Format("15:04:05")
	name := peer.DisplayName()
	log.Printf("%s%s ◆ %sJOIN   %s%s (%s)%s  %s[%s]%s  %s%d peers now%s",
		ts, cMagenta, cBold, cReset, shortPID(peer.PID()), name, cMagenta, cDim, transport, cReset, cBold, peerCount, cReset)
}

// LogUnregister logs a peer disconnecting.
//
//	◇ LEAVE  6b2cab87 (142201026)  2 peers now
func LogUnregister(peer PeerConn, peerCount int) {
	ts := time.Now().Format("15:04:05")
	name := peer.DisplayName()
	log.Printf("%s%s ◇ %sLEAVE  %s%s (%s)%s  %s%d peers now%s",
		ts, cYellow, cBold, cReset, shortPID(peer.PID()), name, cYellow, cBold, peerCount, cReset)
}

// LogPing logs a ping round-trip measurement.
//
//	📡 PING   42ms  b2641982 (malakar)
func LogPing(peer PeerConn, ms int64) {
	ts := time.Now().Format("15:04:05")
	name := peer.DisplayName()
	log.Printf("%s%s 📡 PING %s%4dms%s  %s%s (%s)%s",
		ts, cCyan, cBold, ms, cReset, cReset, shortPID(peer.PID()), name, cReset)
}

// LogSystem logs a system event.
//
//	● UDP listener ready on :9000
func LogSystem(format string, args ...interface{}) {
	ts := time.Now().Format("15:04:05")
	log.Printf("%s%s ● %s%s%s%s", ts, cCyan, cReset, cCyan, fmt.Sprintf(format, args...), cReset)
}

// LogError logs an error.
//
//	⚠ WS read error from b2641982: connection reset
func LogError(format string, args ...interface{}) {
	ts := time.Now().Format("15:04:05")
	log.Printf("%s%s ⚠ %s%s%s%s", ts, cRed, cReset, cRed, fmt.Sprintf(format, args...), cReset)
}

// LogWarn logs a warning.
//
//	! target peer deadbeef not found
func LogWarn(format string, args ...interface{}) {
	ts := time.Now().Format("15:04:05")
	log.Printf("%s%s ! %s%s%s%s", ts, cYellow, cReset, cYellow, fmt.Sprintf(format, args...), cReset)
}

// ── Internal helpers ───────────────────────────────────────────────────

func typeLabel(t uint8) string {
	switch t {
	case TypeACK:
		return "ACK"
	case TypeMedia:
		return "MEDIA"
	case TypeMSG:
		return "MSG"
	case TypeJoin:
		return "JOIN"
	case TypeLeave:
		return "LEAVE"
	case TypeCanvas:
		return "CANVAS"
	case TypePing:
		return "PING"
	case TypePong:
		return "PONG"
	default:
		return fmt.Sprintf("?(%d)", t)
	}
}

func shortPID(pid string) string {
	if len(pid) > 8 {
		return pid[:8]
	}
	return pid
}

func peerLabel(p PeerConn) string {
	name := p.DisplayName()
	if name != "" && name != shortPID(p.PID()) {
		return fmt.Sprintf(" (%s)", name)
	}
	return ""
}

func trimPayload(p []byte) string {
	if len(p) == 0 {
		return "{}"
	}
	if p[0] == '{' || p[0] == '[' {
		s := string(p)
		if len(s) > 80 {
			s = s[:80] + "…"
		}
		return s
	}
	return fmt.Sprintf("%dB binary", len(p))
}

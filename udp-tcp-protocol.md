# Communicate Protocol Specification v1

This document defines the binary packet protocol used between the Communicate
server and any client (React web app, JavaFX desktop app, or raw UDP sender).
The format is transport-agnostic — the same bytes flow over WebSocket frames,
TCP streams, or UDP datagrams.

---

## 1. Transport

| Transport | Port | Typical client |
|---|---|---|
| WebSocket (ws://) | same as HTTP (`:8080` dev) | React / browser JS |
| Raw TCP | TBD | JavaFX, any TCP socket |
| Raw UDP | `:9000` | JavaFX, low-latency media |

**Current implementation:** WebSocket and UDP are active. The same binary
packet format works over both. TCP listener will be added later.

---

## 2. Binary packet format

Every packet has a **4-byte header** followed by a variable-length payload.

```
┌────────────────────────────────────────────────────────────┐
│ Byte 0       │ Version (always 0x01)                       │
│ Byte 1       │ Packet Type                                 │
│ Bytes 2–3    │ Payload Length (uint16, big-endian)         │
│ Bytes 4 … N  │ Payload (0–65535 bytes)                     │
└────────────────────────────────────────────────────────────┘
```

| Field | Size | Description |
|---|---|---|
| Version | 1 B | Protocol version. Current: `0x01`. Reject if unknown. |
| Type | 1 B | Packet type. See §3. |
| Payload Length | 2 B | Number of payload bytes, big-endian. Max 65535 (64 KiB). |
| Payload | N B | Opaque bytes. Interpretation depends on type. |

### Example byte dump — MSG "hello"

```
01 02 00 1C  7B 22 74 6F  22 3A 22 70  65 65 72 2D   │ ....{"to":"peer-
31 32 33 22  2C 22 74 65  78 74 22 3A  22 68 65 6C   │ 123","text":"hel
6C 6F 22 7D                                         │ lo"}
```

- `01` — version 1
- `02` — type = MSG
- `00 1C` — payload length = 28 bytes
- Then 28 bytes of JSON: `{"to":"peer-123","text":"hello"}`

---

## 3. Packet types

| Value | Name | Direction | Purpose |
|---|---|---|---|
| `0x00` | **ACK** | Server → Client | Acknowledge receipt of a packet |
| `0x01` | **Media** | Client → Server → Client(s) | Audio/video frame relay |
| `0x02` | **MSG** | Client → Server → Client(s) | Chat message or signalling (SDP, ICE) |
| `0x03` | **Join** | Server → Client(s) | Peer connected |
| `0x04` | **Leave** | Server → Client | Peer disconnected |
| `0x05` | **Canvas** | Client → Server → Client(s) | Live canvas draw / stroke data |

Types `0x06`–`0x63` (6–99) are reserved for future built-in use.  
Types `0x64`–`0xFF` (100–255) are available for application extensions.

---

## 4. Payload formats by type

### 4.1 ACK (0x00)

**Direction:** Server → Client notification. UDP clients may also send an
empty Leave packet before closing to request immediate unregister.  
**Payload:** Echoes the payload of the packet being acknowledged,
prefixed with the acknowledged packet type.

```json
{"ack_type": 2, "echo": {…}}
```

| Field | Type | Description |
|---|---|---|
| `ack_type` | int | The type of the packet being acknowledged |
| `echo` | any | The first 256 bytes of the original payload (truncated) |

Clients should **not** send ACK packets. The server generates them.

---

### 4.2 MSG (0x02)

**Direction:** Client → Server → Client(s).  
**Payload:** JSON with a `to` field for routing. The server injects `from`.

```json
{
  "to": "<peer-id>",
  "text": "Hello, world!"
}
```

| Field | Required | Description |
|---|---|---|
| `to` | No | Target peer ID. Omit or leave `""` to broadcast to all. |
| `from` | **Server-set** | The server sets this to the sender's peer ID. Clients MUST NOT set it. |
| `text` | No | Chat message body. |

You can add any extra fields — the server relays the full JSON object unchanged
(except for injecting `from`).

**Signalling use case** (WebRTC SDP/ICE exchange):

```json
{
  "to": "<peer-id>",
  "sdp": "v=0\r\no=…",
  "type": "offer"
}
```

```json
{
  "to": "<peer-id>",
  "candidate": "candidate:…",
  "sdpMLineIndex": 0
}
```

---

### 4.3 Media (0x01)

**Direction:** Client → Server → Client(s).  
**Payload:** Raw binary (typically an encoded audio or video frame). The server
**broadcasts** media packets to every other peer in the session.

There is no JSON envelope — the payload is the media frame directly. Routing
is implicit: media flows to all connected peers except the sender.

> **Future:** Once the queue manager is implemented, media packets will be
> routed 1:1 according to the active call state established by prior MSG
> signalling.

---

### 4.4 Join (0x03)

**Direction:** Server → Client only.  
There are two Join variants:

**Welcome (sent ONLY to the newly connected peer):**

```json
{
  "peer_id": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6",
  "peers": ["existing-id-1", "existing-id-2"]
}
```

**Peer list update (sent to EVERY peer on any connect/disconnect):**

```json
{
  "event": "peer_list",
  "peers": [
    {"peer_id": "abc123…", "username": "johndoe"},
    {"peer_id": "def456…", "username": "janedoe"}
  ]
}
```

This is the authoritative list of who is online. Clients should replace their
local peer list with this every time it arrives. The server sends it whenever
any peer joins or leaves.

---

### 4.5 Leave (0x04)

**Direction:** Server → Client only.  

```json
{
  "peer_id": "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6"
}
```

---

### 4.6 Canvas (0x05)

**Direction:** Client → Server → Client(s).  
**Payload:** JSON describing a draw action. The server **broadcasts** canvas
packets to every other peer. The payload structure is application-defined;
below is the expected shape.

```json
{
  "tool": "pen",
  "color": "#ff0000",
  "size": 3,
  "points": [{"x": 100, "y": 200}, {"x": 105, "y": 205}],
  "action": "stroke"
}
```

| Field | Type | Description |
|---|---|---|
| `tool` | string | Drawing tool: `"pen"`, `"eraser"`, `"shape"`, `"text"`, etc. |
| `color` | string | CSS colour string (e.g. `"#ff0000"`) |
| `size` | number | Brush / eraser size in pixels |
| `points` | array | Array of `{x, y}` objects for the stroke path |
| `action` | string | `"stroke"`, `"clear"`, `"undo"`, `"redo"` |

Additional fields can be added as the canvas feature grows. The server relays
the payload as-is (no `from` injection — Canvas payloads are opaque to the
hub, but the sender identity is implicit from the connection).

---

## 5. Connection lifecycle

### 5.1 WebSocket connect

Pass an optional access token to get your username attached:

```
ws://server:8080/ws?token=<access_token>
```

```
Client                          Server                       Other peers
  │                                │                             │
  │── GET /ws?token=xxx ─────────▶│                             │
  │                                │                             │
  │◀── 101 Switching Protocols ───│                             │
  │                                │                             │
  │◀── Join {peer_id, peers[]} ──│  (welcome — your ID + old list)
  │◀── Join {event:peer_list} ───│  (full list with usernames)──▶│
```

The server assigns every peer a 32-character hex ID (16 random bytes).
If a valid token is provided, the peer's display name is set to the DB
username. Otherwise a fallback like `WS-a1b2c3d4` is used.

### 5.2 Send a chat message

```
Client A                        Server                       Client B
  │                                │                             │
  │── MSG {"to":"<B>","text":"hi"}─▶│                             │
  │                                │                             │
  │◀── ACK {ack_type:2} ──────────│                             │
  │                                │── MSG {"from":"A","text":"hi"}──▶│
```

### 5.3 Disconnect

```
Client A                        Server                       Client B
  │                                │                             │
  │── (WS close / UDP LEAVE / UDP timeout) ─▶│                  │
  │                                │                             │
  │                                │── Join {peer_list} ───────▶│
  │                                │   (updated list without A)  │
```

### 5.4 UDP client — full lifecycle

UDP clients don't have a persistent connection. Here is everything you need
to know.

**Step 1 — Send any packet.** The very first datagram you send to
`server:9000` auto-registers you. You can send an empty MSG if you just want
to join:

```
01 02 00 02  {}     →  MSG with empty JSON payload
```

**Step 2 — Receive your peer ID.** The server replies with a **Join**
(type `0x03`) containing your assigned 32-char hex peer ID:

```json
{"peer_id": "f7e8d9c0b1a2…", "peers": ["abc123…", "def456…"]}
```

Store this `peer_id` — other peers use it to send messages to you.

**Step 3 — Receive the peer list.** Immediately after, the server sends a
second Join packet with the **full online list** (including usernames and
avatars):

```json
{
  "event": "peer_list",
  "peers": [
    {"peer_id": "abc123…", "username": "dhruvadeep", "avatar": "https://…", "latency_ms": 15},
    {"peer_id": "def456…", "username": "janedoe",    "avatar": "",          "latency_ms": 42},
    {"peer_id": "f7e8…",   "username": "UDP-f7e8d9c0","avatar": "",          "latency_ms": 0}
  ]
}
```

**Step 4 — The list auto-refreshes.** Every time **any** peer connects,
disconnects, or gets a new ping measurement, the server broadcasts a new
`peer_list` to **everyone**. You don't need to poll — just listen and replace
your local list with each incoming `peer_list`.

**Step 5 — Send messages.** Build your packet (4-byte header + JSON payload)
and send it as a datagram. The server ACKs every packet.

**Step 6 — Reply to server pings.** The server sends PING every 10 seconds.
Clients only echo the same payload back as PONG. Do not start a client-side
ping or heartbeat timer for latency; if a UDP peer stops sending traffic or
PONGs, the server removes the stale entry after a timeout.

**Summary — what you receive:**

| Packet type | When | What it contains |
|---|---|---|
| `0x00` ACK | After every packet you send | Echo of your payload |
| `0x02` MSG | When someone sends you a message | `{"from":"…","text":"…"}` |
| `0x03` Join | On connect/disconnect + ping measurement | `{"event":"peer_list","peers":[{"peer_id":"…","latency_ms":42}]}` |
| `0x01` Media | When someone streams media | Raw binary frame |
| `0x05` Canvas | When someone draws | `{"tool":"pen","points":[…]}` |

That's it. Send datagrams to `server:9000`, read responses from the same
socket, replace your peer list on every `peer_list` event. No polling, no
client-side ping timer, no separate registration step.

---

## 6. Error handling

Errors are sent as MSG packets (type `0x02`) with an `error` field:

```json
{
  "error": "peer not found",
  "peer": "deadbeef"
}
```

```json
{
  "error": "invalid JSON payload"
}
```

The server sends errors **to the sender** of the malformed packet. Malformed
packets (unknown version, bad header) cause the server to **close the
connection**.

---

## 7. Design notes for implementors

### 7.1 Why no sender ID in the header?

The server knows which peer a packet came from — the connection itself carries
that identity. Adding a sender ID to every packet would bloat the header by
16+ bytes. Instead, the server injects `from` into MSG payloads so receivers
can trust the identity.

For Media packets (high-frequency, latency-sensitive), the sender is implicit
from the active call context.

### 7.2 Why uint16 for payload length?

64 KiB per packet is enough for:
- Any chat/signalling JSON (well under 1 KiB)
- Individual audio frames (~50–200 bytes for Opus)
- Individual video frames (usually < 10 KiB for a keyframe at reasonable res)
- ICE candidates and SDP (under 8 KiB)

Larger payloads can be split across multiple packets at the application layer.

### 7.3 Future: queue manager

The server's current Hub routes packets directly (receive → route → send).
The design leaves room to insert a **queue manager** between routing and
transmission that can:

- Prioritise ACKs and signalling over media
- Shape traffic per peer (leaky bucket / token bucket)
- Batch small packets into a single write
- Drop low-priority frames when a peer's buffer is full

The `Peer.Send()` method is the insertion point — it currently enqueues on a
channel; a queue manager would wrap that channel with a multi-queue scheduler.

### 7.4 UDP transport

UDP is active on port **9000** (configurable via `UDP_PORT`). Each datagram
carries exactly one packet. Clients must:

- The first datagram from a new address auto-registers the peer
- Handle lost datagrams (UDP is unreliable)
- Include a sequence number in the JSON payload if ordering matters
- Expect ACKs and Join/Leave packets back on the same port the datagram was sent from

---

## 8. Quick reference — binary header

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
┌───────────────┬───────────────┬───────────────────────────────────┐
│ Version (0x01)│     Type      │    Payload Length (big-endian)    │
└───────────────┴───────────────┴───────────────────────────────────┘
│                                                                   │
│                        Payload bytes …                            │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

Go reference: [internal/comm/packet.go](internal/comm/packet.go) — `Encode()` / `DecodePacket()`.

package signaling

import (
	"net/http"

	"Communicate/internal/comm"
	"Communicate/internal/store/db/queries/sessions"
	"Communicate/internal/store/db/queries/user"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func WebSocket(hub *comm.Hub, pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		displayName := ""
		avatarURL := ""

		if token := r.URL.Query().Get("token"); token != "" {
			s, err := sessions.GetByAccessToken(r.Context(), pool, token)
			if err != nil {
				comm.LogError("WS auth lookup error: %v", err)
			} else if s == nil {
				comm.LogWarn("WS auth: invalid token from %s", r.RemoteAddr)
			} else {
				u, err := user.FindByID(r.Context(), pool, s.UserID)
				if err != nil {
					comm.LogError("WS user lookup error: %v", err)
				} else if u != nil {
					displayName = u.Username
					if u.ProfileImageURL != nil {
						avatarURL = *u.ProfileImageURL
					}
				}
			}
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			comm.LogError("WS upgrade failed: %v", err)
			return
		}

		peer := comm.NewPeer(comm.NewPeerID(), conn)
		if displayName != "" {
			peer.SetDisplayName(displayName)
		}
		if avatarURL != "" {
			peer.SetAvatarURL(avatarURL)
		}

		hub.Register(peer)

		go peer.ReadPump(hub)
		go peer.WritePump()
	}
}

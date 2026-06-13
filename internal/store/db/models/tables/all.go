package tables

import "Communicate/internal/store/db/models/structure"

var All = []structure.Migration{
	Users,
	UserSessions,
}

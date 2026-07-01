package indexes

import "Communicate/internal/store/db/models/structure"

var All = []structure.Index{
	Users,
	UserSessions,
	EmailVerificationTokens,
	PasswordResetTokens,
}

package cache

// UserSync builds the key that records when auth last synced a user's Clerk
// profile into the users table. The value is the sync time as a unix
// timestamp; readers decide how old is too old.
func (kb *KeyBuilder) UserSync(userID string) string {
	return "usersync:" + userID
}

package auth

type Database struct {
	Realm    string
	username string
	password string
	records  map[string]string
}

func newAuthDB(realm string) *Database {
	if realm == "" {
		realm = "dorsvr streaming server"
	}
	return &Database{
		Realm: realm,
	}
}

func (a *Database) InsertUserRecord(username, password string) {
	if username == "" || password == "" {
		return
	}

	_, existed := a.records[username]
	if !existed {
		a.records[username] = password
	}
}

func (a *Database) RemoveUserRecord(username string) {
	_, existed := a.records[username]
	if existed {
		delete(a.records, username)
	}
}

func (a *Database) LookupPassword(username string) (password string) {
	password, _ = a.records[username]
	return
}

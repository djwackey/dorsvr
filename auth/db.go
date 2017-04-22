package auth

type Database struct {
	Realm    string
	username string
	password string
	records  map[string]string
}

func NewAuthDatabase(realm string) *Database {
	if realm == "" {
		realm = "dorsvr streaming server"
	}
	return &Database{
		Realm: realm,
	}
}

func (d *Database) InsertUserRecord(username, password string) {
	if username == "" || password == "" {
		return
	}

	_, existed := d.records[username]
	if !existed {
		d.records[username] = password
	}
}

func (d *Database) RemoveUserRecord(username string) {
	_, existed := d.records[username]
	if existed {
		delete(d.records, username)
	}
}

func (d *Database) LookupPassword(username string) (password string) {
	password, _ = d.records[username]
	return
}

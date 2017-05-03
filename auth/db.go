package auth

// Database stores username and password to implement access control
type Database struct {
	Realm    string
	username string
	password string
	records  map[string]string
}

// NewAuthDatabase returns a pointer to a new instance of authorization database
func NewAuthDatabase(realm string) *Database {
	if realm == "" {
		realm = "dorsvr streaming server"
	}
	return &Database{
		Realm: realm,
	}
}

// InsertUserRecord inserts user record, it contains username and password fields
func (d *Database) InsertUserRecord(username, password string) {
	if username == "" || password == "" {
		return
	}

	_, existed := d.records[username]
	if !existed {
		d.records[username] = password
	}
}

// RemoveUserRecord removes user record
func (d *Database) RemoveUserRecord(username string) {
	_, existed := d.records[username]
	if existed {
		delete(d.records, username)
	}
}

// LookupPassword lookups the password by username
func (d *Database) LookupPassword(username string) (password string) {
	password, _ = d.records[username]
	return
}

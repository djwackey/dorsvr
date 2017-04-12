package rtspserver

import (
	"crypto/md5"
	"fmt"
	"io"
	"strings"
	sys "syscall"
)

const counter = 0

type Authentication struct {
	realm    string
	nonce    string
	username string
	password string
	records  map[string]string
}

func newAuthentication(realm string) *Authentication {
	if realm == "" {
		realm = "dorsvr streaming server"
	}
	return &Authentication{
		realm: realm,
	}
}

func (a *Authentication) insertUserRecord(username, password string) {
	if username == "" || password == "" {
		return
	}

	_, existed := a.records[username]
	if !existed {
		a.records[username] = password
	}
}

func (a *Authentication) removeUserRecord(username string) {
	_, existed := a.records[username]
	if existed {
		delete(a.records, username)
	}
}

func (a *Authentication) lookupPassword(username string) (password string) {
	password, _ = a.records[username]
	return
}

func (a *Authentication) randomNonce() {
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)

	counter++
	seedData := fmt.Sprintf("%d.%06d%d", timeNow.Sec, timeNow.Usec, counter)

	// Use MD5 to compute a 'random' nonce from this seed data:
	h := md5.New()
	io.WriteString(h, seedData)
	a.nonce = string(h.Sum(nil))
}

func (a *Authentication) computeDigestResponse(cmd, url string) string {
	ha1Data := fmt.Sprintf("%s:%s:%s", a.username, a.realm, a.password)
	ha2Data := fmt.Sprintf("%s:%s", cmd, url)

	h1 := md5.New()
	h2 := md5.New()
	io.WriteString(h1, ha1Data)
	io.WriteString(h2, ha2Data)

	digestData := fmt.Sprintf("%s:%s:%s", h1.Sum(nil), a.nonce, h2.Sum(nil))

	h3 := md5.New()
	io.WriteString(h3, digestData)

	return string(h3.Sum(nil))
}

type AuthorizationHeader struct {
	uri      string
	realm    string
	nonce    string
	username string
	response string
}

func parseAuthorizationHeader(buf string) *AuthorizationHeader {
	// First, find "Authorization:"
	for {
		if buf == "" {
			return nil
		}

		if strings.EqualFold(buf[:22], "Authorization: Digest ") {
			break
		}
		buf = buf[1:]
	}

	// Then, run through each of the fields, looking for ones we handle:
	var n1, n2 int
	var parameter, value, username, realm, nonce, uri, response string
	fields := buf[22:]
	for {
		n1, _ = fmt.Sscanf(fields, "%[^=]=\"%[^\"]\"", &parameter, &value)
		n2, _ = fmt.Sscanf(fields, "%[^=]=\"\"", &parameter)
		if n1 != 2 && n2 != 1 {
			break
		}
		if strings.EqualFold(parameter, "username") {
			username = value
		} else if strings.EqualFold(parameter, "realm") {
			realm = value
		} else if strings.EqualFold(parameter, "nonce") {
			nonce = value
		} else if strings.EqualFold(parameter, "uri") {
			uri = value
		} else if strings.EqualFold(parameter, "response") {
			response = value
		}
		fields = fields[len(parameter)+2+len(value)+1:]
		for fields[0] == ' ' || fields[0] == ',' {
			fields = fields[1:]
		}
		if fields == "" || fields[0] == '\r' || fields[0] == '\n' {
			break
		}
	}

	return &AuthorizationHeader{
		uri:      uri,
		realm:    realm,
		nonce:    nonce,
		username: username,
		response: response,
	}
}

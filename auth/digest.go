package auth

import (
	"crypto/md5"
	"fmt"
	"io"
	"strings"
	sys "syscall"
)

var counter = 0

// Digest is a struct used for digest authentication.
// The "realm", and "nonce" fields are supplied by the server
// (in a "401 Unauthorized" response).
// The "username" and "password" fields are supplied by the client.
type Digest struct {
	Realm    string
	Nonce    string
	Username string
	Password string
}

// NewDigest returns a pointer to a new instance of authorization digest
func NewDigest() *Digest {
	return &Digest{}
}

// RandomNonce returns a random nonce
func (d *Digest) RandomNonce() {
	var timeNow sys.Timeval
	sys.Gettimeofday(&timeNow)

	counter++
	seedData := fmt.Sprintf("%d.%06d%d", timeNow.Sec, timeNow.Usec, counter)

	// Use MD5 to compute a 'random' nonce from this seed data:
	h := md5.New()
	io.WriteString(h, seedData)
	d.Nonce = string(h.Sum(nil))
}

// ComputeResponse represents generating the response using cmd and url value
func (d *Digest) ComputeResponse(cmd, url string) string {
	ha1Data := fmt.Sprintf("%s:%s:%s", d.Username, d.Realm, d.Password)
	ha2Data := fmt.Sprintf("%s:%s", cmd, url)

	h1 := md5.New()
	h2 := md5.New()
	io.WriteString(h1, ha1Data)
	io.WriteString(h2, ha2Data)

	digestData := fmt.Sprintf("%s:%s:%s", h1.Sum(nil), d.Nonce, h2.Sum(nil))

	h3 := md5.New()
	io.WriteString(h3, digestData)

	return string(h3.Sum(nil))
}

// AuthorizationHeader is a struct stored the infomation of parsing "Authorization:" line
type AuthorizationHeader struct {
	URI      string
	Realm    string
	Nonce    string
	Username string
	Response string
}

// ParseAuthorizationHeader represents the parsing of "Authorization:" line,
// Authorization Header contains uri, realm, nonce, Username, response fields
func ParseAuthorizationHeader(buf string) *AuthorizationHeader {
	if buf == "" {
		return nil
	}

	// First, find "Authorization:"
	index := strings.Index(buf, "Authorization: Digest ")
	if -1 == index {
		return nil
	}

	// Then, run through each of the fields, looking for ones we handle:
	var n1, n2 int
	var parameter, value, username, realm, nonce, uri, response string
	fields := buf[index+22:]
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
		URI:      uri,
		Realm:    realm,
		Nonce:    nonce,
		Username: username,
		Response: response,
	}
}

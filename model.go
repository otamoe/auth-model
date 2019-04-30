package model

var (
	AuthOrigin   string
	UserOrigin   string
	ClientID     string
	ClientSecret string
)

func Config(authOrigin string, userOrigin string, clientID string, clientSecret string) {
	AuthOrigin = authOrigin
	UserOrigin = userOrigin
	ClientID = clientID
	ClientSecret = clientSecret
}

func Start() {
	initTokenPublicKeys()
}

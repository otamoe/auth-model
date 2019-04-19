package model

var (
	AUTH = ""
	USER = ""
)

func Start(auth string, user string) {
	AUTH = auth
	USER = user
	initToken()
}

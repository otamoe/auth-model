package model

import (
	"io/ioutil"
	"net/http"
	"testing"
)

func Init() {
	Config("http://auth.auth.de:8080", "http://user.auth.de:8080", "5cb2d0ba11ca2b19eefc202b", "wwwwwww")
}
func TestGetClientCredentials(t *testing.T) {
	Init()
	client := GetClientCredentials([]string{"user:all"})

	req, err := http.NewRequest("GET", "http://user.auth.de:8080/me/token/me/", nil)
	if err != nil {
		t.Error(err)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Error(err)
	}
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode >= 300 {
		t.Fatal("Status Code", res.StatusCode, string(b))
	}

	t.Log(string(b))
}

//
// func TestGetAuthorizeURL(t *testing.T) {
// 	Init()
// 	url := GetAuthorizeURL("state", "http://localhost:8082/oauth/authorize", []string{"user:all"})
// 	t.Log(url.String())
// }
//
// func TestGetAuthorizeURL(t *testing.T) {
// 	Init()
// 	url := GetAuthorizeURL("state", "http://localhost:8082/oauth/authorize", []string{"user:all"})
// 	t.Log(url.String())
// }

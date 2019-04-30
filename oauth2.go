package model

import (
	"context"
	"net/http"

	"golang.org/x/oauth2/clientcredentials"
)

func GetClientCredentials(scopes []string) (client *http.Client) {
	conf := clientcredentials.Config{
		ClientID:     ClientID,
		ClientSecret: ClientSecret,
		TokenURL:     AuthOrigin + "/token",
		Scopes:       scopes,
	}
	client = conf.Client(context.Background())
	return
}

// func GetAuthorizeURL(state string, redirect_uri string, scopes []string) (authorizeURL *url.URL) {
// 	conf := &oauth2.Config{
// 		ClientID:     ClientID,
// 		ClientSecret: ClientSecret,
// 		Scopes:       scopes,
// 		Endpoint: oauth2.Endpoint{
// 			TokenURL: AuthOrigin + "/token",
// 			AuthURL:  AuthOrigin + "/authorize",
// 		},
// 	}
//
// 	var err error
// 	if authorizeURL, err = url.Parse(conf.AuthCodeURL(state, oauth2.SetAuthURLParam("redirect_uri", redirect_uri))); err != nil {
// 		panic(err)
// 	}
// 	return
// }
//
// func GetRedirectURL(code string, state string, redirect_uri string) (token *oauth2.Token, err error) {
//
// 	conf := &oauth2.Config{
// 		ClientID:     ClientID,
// 		ClientSecret: ClientSecret,
// 		Endpoint: oauth2.Endpoint{
// 			TokenURL: AuthOrigin + "/token",
// 			AuthURL:  AuthOrigin + "/authorize",
// 		},
// 	}
// 	// token, err = conf.Exchange(ctx, code, oauth2.SetAuthURLParam("state", state), oauth2.SetAuthURLParam("redirect_uri", redirect_uri))
// 	// if err != nil {
// 	// 	log.Fatal(err)
// 	// }
//
// }
//
// // ctx := context.Background()
// //
// // // Use the authorization code that is pushed to the redirect
// // // URL. Exchange will do the handshake to retrieve the
// // // initial access token. The HTTP Client returned by
// // // conf.Client will refresh the token as necessary.
// // var code string
// // if _, err := fmt.Scan(&code); err != nil {
// //     log.Fatal(err)
// // }
// //
// // // Use the custom HTTP client when requesting a token.
// // httpClient := &http.Client{Timeout: 2 * time.Second}
// // ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
// //
// // tok, err := conf.Exchange(ctx, code)
// // if err != nil {
// //     log.Fatal(err)
// // }
// //
// // client := conf.Client(ctx, tok)
// // _ = client

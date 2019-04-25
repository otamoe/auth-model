package model

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/globalsign/mgo"
	"github.com/otamoe/gin-server/errs"
	ginLogger "github.com/otamoe/gin-server/logger"
	"github.com/sirupsen/logrus"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
)

type (
	TokenConfig struct {
		Types    []string
		Required bool
		Expired  bool
		Cache    bool
	}
	TokenClaims struct {
		Name     string        `json:"name"`
		UserID   bson.ObjectId `json:"user_id"`
		Type     string        `json:"type"`
		Scope    string        `json:"scope"`
		Username string        `json:"username"`
		Nickname string        `json:"nickname"`
		jwt.StandardClaims
	}

	TokenPublicKey struct {
		Name           string           `json:"name"`
		Hash           string           `json:"hash"`
		PublicKeyBytes []byte           `json:"public_key"`
		PublicKey      *ecdsa.PublicKey `json:"-"`
	}

	TokenPublicKeys struct {
		Time    time.Time         `json:"-"`
		Results []*TokenPublicKey `json:"results"`
	}

	TokenErrors struct {
		Errors     []*errs.Error `json:"errors,omitempty"`
		StatusCode int           `json:"status_code,omitempty"`
	}
)

var tokenPublicKeys atomic.Value

func TokenMiddleware(c TokenConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		var token *Token
		defer func() {
			if err == nil && c.Required && token == nil {
				err = ErrTokenNotFound
			}
			if err != nil {
				switch err.(type) {
				case *errs.Error:
					err2 := err.(*errs.Error).Clone()
					err2.StatusCode = http.StatusUnauthorized
					err = err2
				}
				ctx.Error(err)
				ctx.Abort()
			} else {
				ctx.Next()
			}
		}()

		vary := ctx.Writer.Header().Get("Vary")
		if vary == "" {
			vary = "Authorization"
		} else {
			vary += ", Authorization"
		}
		ctx.Header("Vary", vary)

		auth := ctx.GetHeader("Authorization")

		// header
		if len(auth) > 7 && strings.ToLower(auth[:7]) == "bearer " {
			token, err = GetToken(ctx, c.Types, strings.TrimSpace(auth[7:]), c.Expired, c.Cache)
		}

		return
	}
}

func requestTokenPublicKeys() (publicKeys *TokenPublicKeys, err error) {
	var request *http.Request
	var response *http.Response

	timeoutContext, timeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer timeoutCancel()
	client := &http.Client{}
	if request, err = http.NewRequest("GET", AUTH+"/keys", nil); err != nil {
		return
	}
	request = request.WithContext(timeoutContext)
	if response, err = client.Do(request); err != nil {
		return
	}
	defer response.Body.Close()
	var bodyBytes []byte
	if bodyBytes, err = ioutil.ReadAll(io.LimitReader(response.Body, 1<<20)); err != nil {
		return
	}
	logrus.Debugf("[TOKEN_KEYS] %d %s", response.StatusCode, string(bodyBytes))

	if response.StatusCode >= http.StatusMultipleChoices {
		err = fmt.Errorf("Token public request error status: %d", response.StatusCode)
	}
	publicKeys = &TokenPublicKeys{}
	if err = json.Unmarshal(bodyBytes, publicKeys); err != nil {
		return
	}

	for _, result := range publicKeys.Results {
		if result == nil {
			err = errors.New("found unknown public key type in ECDSA wrapping")
			return
		}
		var pub interface{}
		if pub, err = x509.ParsePKIXPublicKey(result.PublicKeyBytes); err != nil {
			return
		}
		switch pub.(type) {
		case *ecdsa.PublicKey:
			result.PublicKey = pub.(*ecdsa.PublicKey)
		default:
			err = errors.New("found unknown public key type in ECDSA wrapping")
			return
		}
	}
	publicKeys.Time = time.Now()
	return
}

func requestToken(auth string) (token *Token, err error) {
	var request *http.Request
	var response *http.Response

	timeoutContext, timeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer timeoutCancel()
	client := &http.Client{}
	if request, err = http.NewRequest("GET", USER+"/me/token/me/", nil); err != nil {
		return
	}
	request = request.WithContext(timeoutContext)

	request.Header.Add("Authorization", "Bearer "+auth)
	if response, err = client.Do(request); err != nil {
		return
	}
	defer response.Body.Close()
	var bodyBytes []byte
	if bodyBytes, err = ioutil.ReadAll(io.LimitReader(response.Body, 1<<20)); err != nil {
		return
	}

	logrus.Debugf("[TOKEN] %d %s", response.StatusCode, string(bodyBytes))

	if response.StatusCode >= http.StatusMultipleChoices {
		tokenErrors := &TokenErrors{}
		if err = json.Unmarshal(bodyBytes, tokenErrors); err != nil {
			return
		}
		message := []string{}
		for _, val := range tokenErrors.Errors {
			message = append(message, val.Message)
		}
		err = &errs.Error{
			Message:    strings.Join(message, ", "),
			StatusCode: tokenErrors.StatusCode,
		}
		return
	}
	token = &Token{}
	if err = json.Unmarshal(bodyBytes, token); err != nil {
		return
	}
	return
}

func GetToken(ctx *gin.Context, types []string, val string, expired bool, cache bool) (token *Token, err error) {
	key := CONTEXT_TOKEN

	if value, ok := ctx.Get(key); ok {
		token = value.(*Token)
	}

	claims := &TokenClaims{}
	var jwtToken *jwt.Token
	jwtToken, err = jwt.ParseWithClaims(val, claims, func(token *jwt.Token) (interface{}, error) {
		publicKeys := tokenPublicKeys.Load().(*TokenPublicKeys)
		for _, publicKey := range publicKeys.Results {
			if publicKey.Hash != "" && publicKey.PublicKey != nil && publicKey.Hash == claims.Issuer {
				return publicKey.PublicKey, nil
			}
		}
		return nil, ErrTokenNotFound
	})

	if err != nil {
		err = &errs.Error{
			Err:        err,
			StatusCode: http.StatusForbidden,
		}
		return
	}

	if !jwtToken.Valid || claims.Name != "token" {
		err = ErrTokenNotFound
		return
	}

	if token == nil {
		id := claims.Subject
		// token 写入
		token = &Token{}
		if cache {
			if err = ModelToken.Query(ctx).ID(id).PopulatePath("User", ModelUser.Query(ctx)).One(token); err != nil {
				if err != mgo.ErrNotFound {
					return
				}
			}
		}
		if !token.ID.Valid() {
			if token, err = requestToken(val); err != nil {
				return
			}
			if token.User == nil {
				err = ErrUserNotFound
				return
			}
			token.New(ctx, ModelToken, token, true)
			if err = token.Save(); err != nil && !mgo.IsDup(err) {
				return
			}

			// 用户字段
			user := &User{}
			if err = ModelUser.Query(ctx).ID(token.UserID).One(user); err != nil {
				if err != mgo.ErrNotFound {
					return
				}
				// 创建
				user = token.User
				user.New(ctx, ModelUser, user, true)
			} else {

				user.New(ctx, ModelUser, user, false)
				//  更新
				user.Username = token.User.Username
				user.Nickname = token.User.Nickname
				user.Avatar = token.User.Avatar
				user.Locale = token.User.Locale
				user.Description = token.User.Description
				user.Gender = token.User.Gender
				user.Birthday = token.User.Birthday
				user.CreatedAt = token.User.CreatedAt
				user.UpdatedAt = token.User.UpdatedAt
				token.User = user
			}

			// 更新用户
			if err = user.Save(); err != nil && !mgo.IsDup(err) {
				return
			}
			err = nil
		}

		if token.User == nil {
			err = ErrUserNotFound
			return
		}

		logger := ctx.MustGet(ginLogger.CONTEXT).(*ginLogger.Logger)
		logger.TokenID = token.ID
		logger.UserID = token.UserID
		ctx.Set(key, token)
	}

	if token.ID.Hex() != claims.Subject || token.Type != claims.Type || token.UserID.Hex() != claims.UserID.Hex() {
		err = ErrTokenNotFound
		return
	}

	if len(types) != 0 {
		sort.Strings(types)
		if i := sort.SearchStrings(types, token.Type); i == len(types) || types[i] != token.Type {
			err = ErrTokenNotFound
			return
		}
	}

	if expired && token.ExpiredAt.Before(time.Now()) {
		err = ErrTokenHasExpired
		return
	}

	return
}

func initToken() {
	var err error
	var publicKeys *TokenPublicKeys
	if publicKeys, err = requestTokenPublicKeys(); err != nil {
		panic(err)
	}
	tokenPublicKeys.Store(publicKeys)
	go func() {
		for {
			// 计划更新 每小时
			val, err := requestTokenPublicKeys()
			if err == nil {
				tokenPublicKeys.Store(val)
			} else {
				logrus.Error("[TOKEN_KEYS]", err)
			}
			time.Sleep(time.Hour)
		}
	}()
}

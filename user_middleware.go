package model

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/otamoe/gin-server/errs"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

type (
	UserConfig struct {
		Fetch bool
		Cache bool
	}
)

func UserMiddleware(c UserConfig) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		defer func() {
			if err != nil {
				ctx.Error(err)
				ctx.Abort()
			} else {
				ctx.Next()
			}
		}()

		if userParam := ctx.Param("user"); userParam != "" {
			_, err = GetUser(ctx, userParam, c.Cache, c.Fetch)
		}
	}
}

func GetUser(ctx *gin.Context, val string, cache bool, fetch bool) (user *User, err error) {
	key := "user"
	value, ok := ctx.Get(key)
	if user, ok = value.(*User); ok {

	} else {
		if val == "me" {
			if v, ok := ctx.Get(CONTEXT_TOKEN); ok {
				user = v.(*Token).User
				if user == nil {
					err = ErrUserNotFound
					return
				}
				val = user.ID.Hex()
			} else {
				err = ErrUserNotFound
				return
			}
		} else {
			if !bson.IsObjectIdHex(val) {
				err = ErrUserNotFound
				return
			}
			user = &User{}
			if user.ID == "" && cache {
				if err = ModelUser.Query(ctx).ID(val).One(user); err != nil {
					if err != mgo.ErrNotFound {
						return
					}
					err = nil
				}
			}
			if user.ID == "" && fetch {
				if user, err = requestUser(val); err != nil {
					if err != mgo.ErrNotFound {
						return
					}
					user = &User{}
					err = nil
				} else if cache {
					user.New(ctx, ModelUser, user, true)
					if err = user.Save(); err != nil && !mgo.IsDup(err) {
						return
					}
					err = nil
				}
			}
			if user.ID == "" {
				err = ErrUserNotFound
				return
			}
		}
		ctx.Set("user", user)
	}
	if user.ID.Hex() != val {
		err = ErrUserNotFound
		return
	}
	return
}

func requestUser(val string) (user *User, err error) {
	if UserOrigin == "" {
		err = errors.New("auth-model.UserOrigin variable not configured")
		return
	}
	var request *http.Request
	var response *http.Response

	timeoutContext, timeoutCancel := context.WithTimeout(context.Background(), time.Second*10)
	defer timeoutCancel()
	client := &http.Client{}
	if request, err = http.NewRequest("GET", UserOrigin+"/"+url.QueryEscape(val)+"/", nil); err != nil {
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

	logrus.Debugf("[USER] %d %s", response.StatusCode, string(bodyBytes))

	if response.StatusCode >= http.StatusMultipleChoices {
		userErrors := &Errors{}
		if err = json.Unmarshal(bodyBytes, userErrors); err != nil {
			return
		}
		if userErrors.StatusCode == http.StatusNotFound {
			err = mgo.ErrNotFound
		} else {
			message := []string{}
			for _, val := range userErrors.Errors {
				message = append(message, val.Message)
			}
			err = &errs.Error{
				Message:    strings.Join(message, ", "),
				StatusCode: userErrors.StatusCode,
			}
		}
		return
	}
	user = &User{}
	if err = json.Unmarshal(bodyBytes, user); err != nil {
		return
	}
	return
}

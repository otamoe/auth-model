package model

import (
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/gin-gonic/gin"
)

type (
	UserConfig struct {
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
			_, err = GetUser(ctx, userParam)
		}
	}
}

func GetUser(ctx *gin.Context, val string) (user *User, err error) {
	key := "user"
	if value, ok := ctx.Get(key); ok {
		user = value.(*User)
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
			if err = ModelUser.Query(ctx).ID(val).One(user); err != nil {
				if err == mgo.ErrNotFound {
					err = ErrUserNotFound
				}
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

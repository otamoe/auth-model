package model

import (
	"net/http"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/otamoe/gin-server/errs"
	mgoModel "github.com/otamoe/mgo-model"
)

type User struct {
	mgoModel.DocumentBase `json:"-" bson:"-" binding:"-"`
	ID                    bson.ObjectId `json:"_id" bson:"_id" binding:"required,objectid"`
	Username              string        `json:"username" bson:"username"`
	Nickname              string        `json:"nickname" bson:"nickname"`
	Avatar                string        `json:"avatar,omitempty" bson:"avatar,omitempty"`
	Locale                string        `json:"locale,omitempty" bson:"locale,omitempty"`
	Description           string        `json:"description,omitempty" bson:"description,omitempty"`
	Gender                string        `json:"gender,omitempty" bson:"gender,omitempty"`
	AuthTypes             []string      `json:"auth_types,omitempty" bson:"auth_types,omitempty"`
	Birthday              *time.Time    `json:"birthday,omitempty" bson:"birthday,omitempty"`
	CreatedAt             *time.Time    `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UpdatedAt             *time.Time    `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
}

var (
	ErrUserRequired error = &errs.Error{
		Message:    "User is required",
		Path:       "user",
		Type:       "required",
		StatusCode: http.StatusBadRequest,
	}
	ErrUserHasFound error = &errs.Error{
		Message:    "User has found",
		Path:       "user",
		Type:       "has_found",
		StatusCode: http.StatusForbidden,
	}

	ErrUserIDRequired error = &errs.Error{
		Message:    "User ID is required",
		Path:       "user_id",
		Type:       "required",
		StatusCode: http.StatusBadRequest,
	}

	ErrUserNotFound error = &errs.Error{
		Message:    "User not found",
		Path:       "user",
		Type:       "not_found",
		StatusCode: http.StatusNotFound,
	}

	ErrUserIDNotFound error = &errs.Error{
		Message:    "User ID not found",
		Path:       "user_id",
		Type:       "not_found",
		StatusCode: http.StatusNotFound,
	}
)

var ModelUser = &mgoModel.Model{
	Name:     "users",
	Document: &User{},
	Indexs:   []mgo.Index{},
}

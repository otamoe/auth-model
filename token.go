package model

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/otamoe/gin-server/errs"
	ginResource "github.com/otamoe/gin-server/resource"
	"github.com/otamoe/gin-server/scope"
	mgoModel "github.com/otamoe/mgo-model"
)

type (
	Token struct {
		mgoModel.DocumentBase `json:"-" bson:"-"`
		ID                    bson.ObjectId `json:"_id" bson:"_id"`
		UserID                bson.ObjectId `json:"user_id" bson:"user"`
		User                  *User         `json:"user,omitempty" bson:"-" populate:"UserID"`
		ApplicationID         bson.ObjectId `json:"application_id" bson:"application"`
		ClientID              bson.ObjectId `json:"client_id,omitempty" bson:"client"`
		UserScopes            []*UserScope  `json:"user_scopes,omitempty" bson:"user_scopes,omitempty"`
		CreatedAt             *time.Time    `json:"created_at,omitempty" bson:"created_at"`
		ExpiredAt             *time.Time    `json:"expired_at,omitempty" bson:"expired_at"`
	}
	UserScope struct {
		Scope     *Scope     `json:"scope,omitempty" bson:"scope,omitempty"`
		ExpiredAt *time.Time `json:"expired_at,omitempty" bson:"expired_at,omitempty"`
	}
	Scope struct {
		ApplicationID bson.ObjectId `json:"application_id,omitempty" bson:"application"`
		Level         int           `json:"level" bson:"level"`
		Roles         []ScopeRole   `json:"roles,omitempty" bson:"roles,omitempty"`
	}

	ScopeRole struct {
		Auths  []string               `json:"auths,omitempty" bson:"auths,omitempty" binding:"max=16,dive,required,trim"`
		Status string                 `json:"status,omitempty" bson:"status" binding:"required,oneof=pending approved banned"`
		User   string                 `json:"user,omitempty" bson:"user" binding:"required"`
		Type   string                 `json:"type,omitempty" bson:"type" binding:"required,max=32"`
		Action string                 `json:"action,omitempty" bson:"action" binding:"required,max=32"`
		Params map[string]interface{} `json:"params,omitempty" bson:"params,omitempty"`
	}

	SortScopes []*Scope
)

var (
	CONTEXT_TOKEN = scope.CONTEXT

	ErrTokenRequired error = &errs.Error{
		Message:    "Token is required",
		Path:       "access_token",
		Type:       "required",
		StatusCode: http.StatusBadRequest,
	}

	ErrTokenNotFound error = &errs.Error{
		Message:    "Token not found",
		Path:       "access_token",
		Type:       "not_found",
		StatusCode: http.StatusUnauthorized,
	}
	ErrTokenHasExpired error = &errs.Error{
		Message:    "Token has expired",
		Path:       "access_token",
		Type:       "has_expired",
		StatusCode: http.StatusUnauthorized,
	}
)

var ModelToken = &mgoModel.Model{
	Name:     "tokens",
	Document: &Token{},
	Indexs: []mgo.Index{
		mgo.Index{
			Key:        []string{"user"},
			Background: true,
		},
		mgo.Index{
			Key:        []string{"created_at"},
			Background: true,
		},
		mgo.Index{
			Key:        []string{"expired_at"},
			Background: true,
		},
	},
}

func (token *Token) ValidateScope(resource *ginResource.Resource) (params map[string]interface{}, err error) {
	now := time.Now()
	scopes := SortScopes{}

	ownerID := resource.GetOwner()
	for _, userScope := range token.UserScopes {
		if userScope == nil {
			continue
		}

		// 过期
		if userScope.ExpiredAt != nil && userScope.ExpiredAt.Before(now) {
			continue
		}

		scope := userScope.Scope

		// 权限是空
		if scope == nil {
			continue
		}

		// 应用不同
		if scope.ApplicationID != resource.Application {
			continue
		}
		scopes = append(scopes, scope)
	}

	sort.Sort(scopes)

	authTypes := []string{}
	if token.User != nil && token.User.AuthTypes != nil {
		authTypes = token.User.AuthTypes
	}
	sort.Strings(authTypes)

	for _, scope := range scopes {
		for _, scopeRole := range scope.Roles {
			// 规则没使用
			if scopeRole.Status == "pending" {
				continue
			}

			// 动作
			if scopeRole.Action != resource.Action && scopeRole.Action != "*" {
				continue
			}

			// 认证类型
			if len(scopeRole.Auths) != 0 {
				//  auth type 不匹配
				var authMatch bool
				for _, auth := range scopeRole.Auths {
					// 匹配到了
					if i := sort.SearchStrings(authTypes, auth); i != len(authTypes) && authTypes[i] == auth {
						authMatch = true
						break
					}
				}
				if !authMatch {
					continue
				}
			}

			// 用户
			if scopeRole.User == "" {
				continue
			} else if scopeRole.User == "*" {

			} else if !ownerID.Valid() {
				continue
			} else if scopeRole.User == "me" {
				if token.UserID != ownerID {
					continue
				}
			} else if scopeRole.User != ownerID.Hex() {
				continue
			}

			// 资源
			if scopeRole.Type == "*" || scopeRole.Type == resource.Type {

			} else if matched, _ := regexp.MatchString("^"+strings.Replace(regexp.QuoteMeta(scopeRole.Type), "\\*", ".*", -1)+"$", resource.Type); !matched {
				continue
			}

			if scopeRole.Status == "approved" {
				params = scopeRole.Params
				return
			}
			break
		}
	}

	errParams := bson.M{"action": resource.Action, "type": resource.Type, "application_id": resource.Application}

	if ownerID.Valid() {
		errParams["owner_id"] = ownerID
	}
	err = &errs.Error{
		Message:    fmt.Sprintf("Validate Scope"),
		Type:       "scope",
		Value:      resource.Value,
		StatusCode: http.StatusForbidden,
		Params:     errParams,
	}
	return
}

func (scopes SortScopes) Len() int {
	return len(scopes)
}

func (scopes SortScopes) Swap(i, j int) {
	scopes[i], scopes[j] = scopes[j], scopes[i]
}

func (scopes SortScopes) Less(i, j int) bool {
	return scopes[i].Level > scopes[j].Level
}

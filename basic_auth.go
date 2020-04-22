package wine

import (
	"context"
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/gopub/log"
)

// BasicAuth returns a basic auth interceptor
func BasicAuth(userToPassword map[string]string, realm string) HandlerFunc {
	if len(userToPassword) == 0 {
		log.Panic("userToPassword is empty")
	}

	userToAuthInfo := make(map[string]string)
	for user, password := range userToPassword {
		if len(user) == 0 || len(password) == 0 {
			log.Panic("Empty user or password")
		}
		info := user + ":" + password
		userToAuthInfo[user] = "Basic " + base64.StdEncoding.EncodeToString([]byte(info))
	}

	return func(ctx context.Context, req *Request, next Invoker) Responsible {
		a := req.Authorization()
		for user, info := range userToAuthInfo {
			if info == a {
				ctx = context.WithValue(ctx, CKBasicAuthUser, user)
				return next(ctx, req)
			}
		}
		return RequireBasicAuth(realm)
	}
}

func RequireBasicAuth(realm string) Responsible {
	return ResponsibleFunc(func(ctx context.Context, w http.ResponseWriter) {
		a := "Basic realm=" + strconv.Quote(realm)
		w.Header().Set("WWW-Authenticate", a)
		w.WriteHeader(http.StatusUnauthorized)
	})
}

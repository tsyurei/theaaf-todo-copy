package api

import (
	"encoding/base64"
	"encoding/json"

	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/NYTimes/gziphandler"
	"github.com/gorilla/mux"

	"todo-app/internal/app"
	"todo-app/internal/model"
)

type statusCodeRecorder struct {
	http.ResponseWriter
	http.Hijacker
	StatusCode int
}

func (r *statusCodeRecorder) WriteHeader(statusCode int) {
	// TODO: experiment with this more
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

type API struct {
	App    *app.App
	Config *Config
}

func New(a *app.App) (api *API, err error) {
	api = &API{App: a}
	api.Config, err = InitConfig()
	if err != nil {
		return nil, err
	}
	return api, nil
}

func (api *API) Init(r *mux.Router) {
	r.Handle("/hello", gziphandler.GzipHandler(api.handler(api.RootHandler))).Methods("GET")
	r.Handle("/user/", api.handler(api.CreateUser)).Methods("POST")

	// todo methods
	todosRouter := r.PathPrefix("/todos").Subrouter()
	todosRouter.Handle("/", api.handler(api.GetTodos)).Methods("GET")
	todosRouter.Handle("/", api.handler(api.CreateTodo)).Methods("POST")
	todosRouter.Handle("/{id:[0-9]+}/", api.handler(api.GetTodoById)).Methods("GET")
	todosRouter.Handle("/{id:[0-9]+}/", api.handler(api.UpdateTodoById)).Methods("PATCH")
	todosRouter.Handle("/{id:[0-9]+}/", api.handler(api.DeleteTodoById)).Methods("DELETE")
}

func (api *API) handler(f func(*app.Context, http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

		beginTime := time.Now()

		hijacker, _ := w.(http.Hijacker)

		w = &statusCodeRecorder{
			ResponseWriter: w,
			Hijacker:       hijacker,
		}

		ctx := api.App.NewContext().WithRemoteAddress(api.IPAddressForRequest(r))
		ctx = ctx.WithLogger(ctx.Logger.WithField("request_id", base64.RawURLEncoding.EncodeToString(model.NewId())))

		if username, password, ok := r.BasicAuth(); ok {
			user, err := api.App.GetUserByEmail(username)

			if user == nil || err != nil {
				if err != nil {
					ctx.Logger.WithError(err).Error("unable to get user")
				}
				http.Error(w, "invalid credentials", http.StatusForbidden)
				return
			}

			if ok := user.CheckPassword(password); !ok {
				http.Error(w, "invalid credentials", http.StatusForbidden)
				return
			}

			ctx = ctx.WithUser(user)
		}

		defer func() {
			statusCode := w.(*statusCodeRecorder).StatusCode
			if statusCode == 0 {
				statusCode = 200
			}
			duration := time.Since(beginTime)

			logger := ctx.Logger.WithFields(logrus.Fields{
				"duration":   duration,
				"statusCode": statusCode,
				"remote":     ctx.RemoteAddress,
			})

			logger.Info(r.Method + " " + r.URL.RequestURI())
		}()

		defer func() {
			if r := recover(); r != nil {
				ctx.Logger.Error(fmt.Errorf("%v: %s", r, debug.Stack()))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}

		}()

		w.Header().Set("Content-Type", "application/json")

		if err := f(ctx, w, r); err != nil {
			if verr, ok := err.(*app.ValidationError); ok {
				data, err := json.Marshal(verr)
				if err == nil {
					w.WriteHeader(http.StatusBadRequest)
					_, err = w.Write(data)
				}

				if err != nil {
					ctx.Logger.Error(err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			} else if uerr, ok := err.(*app.UserError); ok {
				data, err := json.Marshal(uerr)
				if err == nil {
					w.WriteHeader(uerr.StatusCode)
					_, err = w.Write(data)
				}

				if err != nil {
					ctx.Logger.Error(err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			} else {
				ctx.Logger.Error(err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}
	})

}

func (api *API) RootHandler(ctx *app.Context, w http.ResponseWriter, r *http.Request) error {
	_, err := w.Write([]byte("{'hello': 'world'}"))
	return err
}

func (api *API) IPAddressForRequest(r *http.Request) string {
	// TODO: experiment more on this
	addr := r.RemoteAddr

	if api.Config.ProxyCount > 0 {
		h := r.Header.Get("X-Forwarded-For")
		if h != "" {
			clients := strings.Split(h, ",")
			if api.Config.ProxyCount > len(clients) {
				addr = clients[0]
			} else {
				addr = clients[len(clients)-api.Config.ProxyCount]
			}
		}
	}

	return strings.Split(strings.TrimSpace(addr), ":")[0]
}

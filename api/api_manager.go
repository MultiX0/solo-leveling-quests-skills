package api

import (
	"log"
	"net/http"
	"time"

	supa "github.com/MultiX0/solo_leveling_system/handler"
	"github.com/MultiX0/solo_leveling_system/handler/quests"
	"github.com/gorilla/mux"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

type ApiServer struct {
	addr string
}

func NewServer(addr string) *ApiServer {
	return &ApiServer{
		addr: addr,
	}
}

func (s *ApiServer) RunServer() error {
	router := mux.NewRouter()

	subrouter := router.PathPrefix("/api/v1").Subrouter()

	supabaseHandler := supa.GetSupabaseHandler()
	supabaseHandler.HandleRequests(subrouter)

	questsHandler := quests.GetNewQuestsHandler()
	questsHandler.RoutesHandler(subrouter)

	middlewareChain := MiddleWareChain(
		LoggerMiddleWare,
	)

	server := http.Server{Addr: s.addr, Handler: middlewareChain(router)}

	log.Println("server running at ", s.addr)

	return server.ListenAndServe()

}

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func LoggerMiddleWare(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		start := time.Now()

		wrapped := &wrappedWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		ip := r.Header.Get("X-Real-IP")
		if ip == "" {
			ip = r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = r.RemoteAddr
			}
		}

		next.ServeHTTP(wrapped, r)

		var color string

		if wrapped.statusCode >= 200 && wrapped.statusCode <= 300 {
			color = Green
		} else {
			color = Red
		}

		log.Printf("%s %s %d %s %s %s %s %s %v", color, "[", wrapped.statusCode, r.Method, "]", Reset, ip, r.URL.Path, time.Since(start))
	}

}

type MiddleWare func(http.Handler) http.HandlerFunc

func MiddleWareChain(middlewares ...MiddleWare) MiddleWare {
	return func(next http.Handler) http.HandlerFunc {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}

		return next.ServeHTTP
	}
}

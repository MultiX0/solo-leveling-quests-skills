package api

import (
	"log"
	"net/http"

	supa "github.com/MultiX0/solo_leveling_system/handler"
	"github.com/MultiX0/solo_leveling_system/handler/quests"
	"github.com/gorilla/mux"
)

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

func LoggerMiddleWare(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("method %s, path %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
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

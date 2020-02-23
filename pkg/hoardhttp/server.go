package hoardhttp

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	n *hoard.Node
	r chi.Router
}

func New(n *hoard.Node, uiPath string) *Server {
	s := &Server{n: n}
	r := chi.NewRouter()

	// manifests
	r.Get("/m/{mID:\\d+}", s.getManifest)
	r.Get("/suggest/{mID:\\d+}", s.suggestTags)
	r.Post("/query", s.queryManifests)

	// data
	r.Get("/d/{mID:\\d+}", s.getData)
	r.Get("/d/{mID:\\d+}/{p}", s.getData)

	// peers
	r.Get("/peers/{peerID}", s.getPeer)
	r.Get("/peers", s.listPeers)
	r.Put("/peers", s.putPeer)
	r.Delete("/peers", s.deletePeer)

	// status
	r.Get("/status", s.status)

	// ui
	if uiPath != "" {
		log.Info("serving ui from ", uiPath)
		uiHandler := http.FileServer(http.Dir(uiPath))
		uiHandler = http.StripPrefix("/ui/", uiHandler)
		r.Mount("/ui", uiHandler)
	}

	s.r = r
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func httpSuccess(w http.ResponseWriter, x interface{}) {
	data, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	_, err = w.Write(data)
	if err != nil {
		log.Error(err)
	}
}

func httpErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	log.Println(err)

	status := http.StatusInternalServerError
	switch err {
	case os.ErrNotExist:
		status = http.StatusNotFound
	}

	w.WriteHeader(status)
	w.Write([]byte("error: " + err.Error()))
	return true
}

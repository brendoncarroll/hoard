package hoard

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
)

type HTTPAPI struct {
	n *Node
	r chi.Router
}

func newHTTPAPI(n *Node) *HTTPAPI {
	s := &HTTPAPI{n: n}
	r := chi.NewRouter()

	r.Get("/m/{mID:\\d+}", s.getManifest)
	r.Get("/m", s.queryManifests)
	r.Get("/d/{mID:\\d+}", s.getData)
	r.Get("/d/{mID:\\d+}/{p}", s.getData)
	r.Get("/status", s.status)

	if n.getUIPath() != "" {
		log.Info("serving ui from ", n.getUIPath())
		uiHandler := http.FileServer(http.Dir(n.getUIPath()))
		uiHandler = http.StripPrefix("/ui/", uiHandler)
		r.Mount("/ui", uiHandler)
	}

	s.r = r
	return s
}

func (s *HTTPAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func (s *HTTPAPI) renderAll(w http.ResponseWriter, r *http.Request) {
	// TODO: build an actual UI
	ctx := r.Context()
	ids, err := s.n.QueryManifests(ctx, nil, 10)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	mfs := []*Manifest{}
	for _, id := range ids {
		mf, err := s.n.GetManifest(ctx, id)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mfs = append(mfs, mf)
	}
	data, _ := json.MarshalIndent(mfs, "", " ")
	w.Write(data)
}

func (s *HTTPAPI) getManifest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mID, err := strconv.Atoi(chi.URLParam(r, "mID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	mf, err := s.n.GetManifest(ctx, uint64(mID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data, _ := json.Marshal(mf)
	w.Write(data)
}

func (s *HTTPAPI) queryManifests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	results, err := s.n.QueryManifests(ctx, nil, 100)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data, _ := json.Marshal(results)
	w.Write(data)
}

func (s *HTTPAPI) getData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mID, err := strconv.Atoi(chi.URLParam(r, "mID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	content, err := s.n.GetData(ctx, uint64(mID), "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ext, _ := s.n.GetTag(ctx, uint64(mID), "extension")
	http.ServeContent(w, r, ext, time.Time{}, content)
}

func (s *HTTPAPI) status(w http.ResponseWriter, r *http.Request) {
	status := s.n.Status()
	data, _ := json.MarshalIndent(status, "", " ")
	w.Write(data)
}

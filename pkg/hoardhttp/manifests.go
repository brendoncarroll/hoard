package hoardhttp

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/brendoncarroll/hoard/pkg/tagdb"
	"github.com/go-chi/chi"
)

func (s *Server) getManifest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mID, err := strconv.Atoi(chi.URLParam(r, "mID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	mf, err := s.n.GetManifest(ctx, uint64(mID))
	if httpErr(w, err) {
		return
	}

	data, _ := json.Marshal(mf)
	w.Write(data)
}

func (s *Server) queryManifests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	q := tagdb.Query{}
	if err := json.Unmarshal(data, &q); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	results, err := s.n.QueryManifests(ctx, q)
	if httpErr(w, err) {
		return
	}
	httpSuccess(w, results)
}

func (s *Server) getData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mID, err := strconv.Atoi(chi.URLParam(r, "mID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	content, err := s.n.GetData(ctx, uint64(mID), "")
	if httpErr(w, err) {
		return
	}

	ext, _ := s.n.GetTag(ctx, uint64(mID), "extension")
	http.ServeContent(w, r, ext, time.Time{}, content)
}

func (s *Server) suggestTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mID, err := strconv.Atoi(chi.URLParam(r, "mID"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	tagSet, err := s.n.SuggestTags(ctx, uint64(mID))
	if httpErr(w, err) {
		return
	}
	data, _ := json.Marshal(tagSet)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

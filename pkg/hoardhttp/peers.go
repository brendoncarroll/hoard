package hoardhttp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/go-chi/chi"
)

func (s *Server) listPeers(w http.ResponseWriter, r *http.Request) {
	ids, err := s.n.ListPeers(r.Context())
	if httpErr(w, err) {
		return
	}
	httpSuccess(w, ids)
}

func (s *Server) getPeer(w http.ResponseWriter, r *http.Request) {
	id, err := peerIDFromReq(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	pinfo, err := s.n.GetPeer(r.Context(), id)
	if httpErr(w, err) {
		return
	}
	if pinfo == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	httpSuccess(w, pinfo)
}

func (s *Server) putPeer(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	peerInfo := &hoard.PeerInfo{}
	if err := dec.Decode(peerInfo); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err := s.n.PutPeer(r.Context(), peerInfo)
	if httpErr(w, err) {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) deletePeer(w http.ResponseWriter, r *http.Request) {
	id, err := peerIDFromReq(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = s.n.DeletePeer(r.Context(), id)
	if httpErr(w, err) {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	status := s.n.Status()
	data, _ := json.MarshalIndent(status, "", " ")
	w.Write(data)
}

func peerIDFromReq(r *http.Request) (p2p.PeerID, error) {
	b64str := chi.URLParam(r, "peerID")
	if base64.URLEncoding.DecodedLen(len(b64str)) != len(p2p.PeerID{}) {
		return p2p.PeerID{}, errors.New("could not parse peer id")
	}
	idBytes, err := base64.URLEncoding.DecodeString(b64str)
	if err != nil {
		return p2p.PeerID{}, err
	}
	id := p2p.PeerID{}
	copy(id[:], idBytes)
	return id, nil
}

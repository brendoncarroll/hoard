package hoard

import (
	"context"

	"github.com/blobcache/blobcache/pkg/blobcache"
	"github.com/blobcache/blobcache/pkg/blobs"
)

var _ blobs.Store = &Store{}

type Store struct {
	bc   blobcache.API
	psID blobcache.PinSetID
}

func newStore(bc blobcache.API, psID blobcache.PinSetID) *Store {
	return &Store{bc: bc, psID: psID}
}

func (s *Store) GetF(ctx context.Context, id blobs.ID, fn func([]byte) error) error {
	return s.bc.GetF(ctx, id, fn)
}

func (s *Store) Post(ctx context.Context, data []byte) (blobs.ID, error) {
	return s.bc.Post(ctx, s.psID, data)
}

func (s *Store) Delete(ctx context.Context, id blobs.ID) error {
	return s.bc.Unpin(ctx, s.psID, id)
}

func (s *Store) Exists(ctx context.Context, id blobs.ID) (bool, error) {
	err := s.bc.GetF(ctx, id, func([]byte) error {
		return nil
	})
	if err == blobs.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) List(ctx context.Context, prefix []byte, ids []blobs.ID) (int, error) {
	panic("not implemented")
}

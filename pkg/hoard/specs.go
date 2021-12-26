package hoard

import (
	"path/filepath"

	"github.com/brendoncarroll/go-state/cadata"
	"github.com/brendoncarroll/go-state/cadata/fsstore"
	"github.com/brendoncarroll/go-state/cells"
	"github.com/brendoncarroll/go-state/cells/httpcell"
	"github.com/brendoncarroll/go-state/posixfs"
	"github.com/gotvc/got/pkg/gotfs"
	"github.com/pkg/errors"

	"github.com/brendoncarroll/hoard/pkg/filecell"
)

type VolumeSpec struct {
	Cell  CellSpec  `json:"cell"`
	Store StoreSpec `json:"store"`
}

type CellSpec struct {
	File *string       `json:"file,omitempty"`
	HTTP *HTTPCellSpec `json:"http,omitempty"`
}

type HTTPCellSpec struct {
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type StoreSpec struct {
	LocalDir *string `json:"local_dir,omitempty"`
}

func MakeVolume(spec VolumeSpec) (*Volume, error) {
	cell, err := MakeCell(spec.Cell)
	if err != nil {
		return nil, err
	}
	store, err := MakeStore(spec.Store)
	if err != nil {
		return nil, err
	}
	return &Volume{
		Cell:  cell,
		Store: store,
	}, nil
}

func MakeCell(spec CellSpec) (cells.Cell, error) {
	switch {
	case spec.File != nil:
		p, err := filepath.Abs(*spec.File)
		if err != nil {
			return nil, err
		}
		return filecell.New(posixfs.NewOSFS(), filepath.ToSlash(p)), nil
	case spec.HTTP != nil:
		return httpcell.New(httpcell.Spec{
			URL:     spec.HTTP.URL,
			Headers: spec.HTTP.Headers,
		}), nil
	default:
		return nil, errors.Errorf("empty cell spec")
	}
}

func MakeStore(spec StoreSpec) (cadata.Store, error) {
	switch {
	case spec.LocalDir != nil:
		fs := posixfs.NewDirFS(*spec.LocalDir)
		return fsstore.New(fs, cadata.DefaultHash, gotfs.DefaultMaxBlobSize), nil
	default:
		return nil, errors.Errorf("empty store spec")
	}
}

package hoardcmd

import (
	"github.com/brendoncarroll/hoard/pkg/hoard"
	"github.com/brendoncarroll/hoard/pkg/indexers/hidx_audio"
)

func DefaultIndexers() map[string]hoard.Indexer {
	return map[string]hoard.Indexer{
		"id3v1": hidx_audio.IndexID3v1,
		"id3v2": hidx_audio.IndexID3v2,
		"flac":  hidx_audio.IndexFLAC,
	}
}

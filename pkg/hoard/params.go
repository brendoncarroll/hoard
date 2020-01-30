package hoard

import (
	"crypto/ed25519"
	"crypto/rand"
	"log"
	"path/filepath"

	"github.com/brendoncarroll/go-p2p/aggswarm"
	"github.com/brendoncarroll/go-p2p/simplemux"
	"github.com/brendoncarroll/go-p2p/sshswarm"
	bolt "go.etcd.io/bbolt"
)

type Params struct {
	Mux simplemux.Muxer
	DB  *bolt.DB

	BlobcacheDB *bolt.DB
	Capacity    uint64
	SourcePaths []string
}

func DefaultParams(dirpath string) (*Params, error) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	// setup database
	db, err := bolt.Open(filepath.Join(dirpath, "hoard.db"), 0644, nil)
	if err != nil {
		return nil, err
	}
	log.Println("connected to db: ", db.Path())

	// setup blobcache database
	bdb, err := bolt.Open(filepath.Join(dirpath, "blobcache.db"), 0644, nil)
	if err != nil {
		return nil, err
	}

	// setup swarm
	swarm1, err := sshswarm.New(":", privKey)
	if err != nil {
		return nil, err
	}
	transports := map[string]aggswarm.Transport{
		"ssh": swarm1,
	}
	swarm := aggswarm.New(privKey, transports)
	mux := simplemux.MultiplexSwarm(swarm)

	return &Params{
		Mux:         mux,
		DB:          db,
		BlobcacheDB: bdb,
		Capacity:    1e5, // about 6 GB
	}, nil
}

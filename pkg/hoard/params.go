package hoard

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/brendoncarroll/go-p2p"
	"github.com/brendoncarroll/go-p2p/s/multiswarm"
	"github.com/brendoncarroll/go-p2p/s/natswarm"
	"github.com/brendoncarroll/go-p2p/s/quicswarm"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

type Params struct {
	Swarm p2p.SecureAskSwarm
	DB    *bolt.DB

	BlobcacheDB *bolt.DB
	Capacity    uint64
	SourcePaths []string
}

func DefaultParams(dirpath string, sourcePaths []string) (*Params, error) {
	pkFilename := "hoard_private_key.pem"
	pkPath := filepath.Join(dirpath, pkFilename)

	var privKey p2p.PrivateKey
	_, err := os.Stat(pkPath)
	if os.IsNotExist(err); err != nil {
		log.Info("private key not found creating at ", pkPath)
		_, privKey, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			panic(err)
		}
		pemData := marshalPrivate(privKey)
		if err := ioutil.WriteFile(pkPath, pemData, 0644); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	log.Println("found private key file")
	keyPem, err := ioutil.ReadFile(pkPath)
	if err != nil {
		return nil, err
	}
	if privKey, err = parsePrivate(keyPem); err != nil {
		return nil, err
	}

	// setup database
	db, err := bolt.Open(filepath.Join(dirpath, "hoard.db"), 0644, nil)
	if err != nil {
		return nil, err
	}
	log.Info("connected to db", db.Path())

	// setup blobcache database
	bdb, err := bolt.Open(filepath.Join(dirpath, "blobcache.db"), 0644, nil)
	if err != nil {
		return nil, err
	}

	// setup swarm
	swarm1, err := setupSwarm(privKey)
	if err != nil {
		return nil, err
	}

	return &Params{
		Swarm:       swarm1.(p2p.SecureAskSwarm),
		DB:          db,
		BlobcacheDB: bdb,
		Capacity:    1e5, // about 6 GB
		SourcePaths: sourcePaths,
	}, nil
}

func setupSwarm(privKey p2p.PrivateKey) (p2p.Swarm, error) {
	s1, err := quicswarm.New("0.0.0.0:", privKey)
	if err != nil {
		return nil, err
	}
	s2 := natswarm.WrapSecureAsk(s1)
	s3 := multiswarm.NewSecureAsk(map[string]p2p.SecureAskSwarm{
		"quic": s2,
	})
	return s3, nil
}

func parsePrivate(pemData []byte) (p2p.PrivateKey, error) {
	pemBlock, _ := pem.Decode(pemData)
	if pemBlock == nil {
		return nil, errors.New("could not parse pem")
	}
	pk, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return pk.(p2p.PrivateKey), nil
}

func marshalPrivate(pk p2p.PrivateKey) []byte {
	data, err := x509.MarshalPKCS8PrivateKey(pk)
	if err != nil {
		panic(err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: data,
	})
	return pemData
}

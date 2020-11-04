module github.com/brendoncarroll/hoard

go 1.15

require (
	github.com/alangpierce/go-forceexport v0.0.0-20160317203124-8f1d6941cd75 // indirect
	github.com/blobcache/blobcache v0.0.0-20201018013000-87bb905b063f
	github.com/brendoncarroll/go-p2p v0.0.0-20201022031311-0751a14857a5
	github.com/brianolson/cbor_go v1.0.0
	github.com/dhowden/tag v0.0.0-20191122115059-7e5c04feccd8
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/marten-seemann/chacha20 v0.2.0 // indirect
	github.com/mewkiz/flac v1.0.6
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.6.1
	go.etcd.io/bbolt v1.3.4
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e
)

replace github.com/shirou/gopsutil => github.com/shirou/gopsutil v3.20.10+incompatible

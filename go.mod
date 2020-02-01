module github.com/brendoncarroll/hoard

go 1.13

require (
	github.com/boltdb/bolt v1.3.1
	github.com/brendoncarroll/blobcache v0.0.0-20200129223006-4aa9bd09981c
	github.com/brendoncarroll/go-p2p v0.0.0-20200131231050-c057f1764318
	github.com/brendoncarroll/webfs v0.0.0-20200130024552-778ba6701fd9
	github.com/dhowden/tag v0.0.0-20191122115059-7e5c04feccd8
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/google/martian v2.1.0+incompatible
	github.com/mewkiz/flac v1.0.6
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cobra v0.0.5
	github.com/stretchr/testify v1.4.0
	go.etcd.io/bbolt v1.3.3
)

replace github.com/brendoncarroll/go-p2p => ../go-p2p

replace github.com/brendoncarroll/blobcache => ../blobcache

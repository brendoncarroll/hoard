# Hoard

Hoard is a tag oriented content management system built on top of the [blobcache](https://github.com/blobcache/blobcache) storage network using the file and directory structures from [WebFS](https://github.com/brendoncarroll/webfs).

Hoard aims to let normal non-technical users organize content, share files, and coordinate group archival.

Hoard is *only* for content you want to share with peers.
**DO NOT** use Hoard for personal data, or data that you are not allowed to distribute.

## How It Works
When a file is imported into Hoard, its data is encrypted using a convergent key and stored in blobcache.
Metadata linking to the data blobs is also created and encrypted.
All this functionality is provided by WebFS.
WebFS spits out a small amount of data (called a `WebRef`) which is a reference, transitively, to this entire structure.

Next, Hoard creates a `Manifest` with the `WebRef` and adds some default tags, like the filename.
Hoard will even suggest tags extracted from common formats like mp3 and flac.
A Hoard Node makes all its manifests searchable by tag to its one hop peers.

Although blobcache blobs are available to anyone who requests them, they are encrypted and cannot be decrypted without a `Manifest`.
This means that the content available to you is the content on your and your peers' nodes. But when it comes time to pull that content down to your node, it is pulled from the blobcache network as a whole, not just the node that gave you the manifest.

You could also get a manifest via another channel like email, or instant messaging. You would be able to pull it down to your node, even if your one-hop nodes didn't have it, as long as it was reachable through blobcache.

Blobcache allows Hoard to only focus on one problem: indexing and organizing.
You can index more content than you can physically store on your device, as long as blobcache can find storage with your peers.
It's not magic though, you are limited by the size of the index.


## Getting Started
All you need is Docker to get started.

First build the docker image
```
make docker
```

Then run it.
```
docker run -it --rm --net=host hoard:latest
```

Now open up a browser and check the status. `http://127.0.0.1:6026/status`.
If you see a pretty-printed JSON object with information about the node, then it's working.

- Hoard exposes it's UI on `localhost:8026` by default.
- The docker image stores all Hoard's data in `/data`
- The docker image will not duplicate data in `/content`.
You still have to tell hoard to import the data to make it searchable, but blobcache will not store any blobs which can be derived from the files in `/content`.

Alternatively if you have Go setup. You can `go run ./cmd/hoard` to spin up a server.
Hoard will create its database and initialize blobcache in the current working directory.
This is the recommended way to work on Hoard.

### Things to Try

To list some manifests:
```
curl -X POST -d '{"limit": 10}' 127.0.0.1:6026/query
```

## Development
### Node
The hoard node process is written in Go.
The entrypoint is in `cmd/`, but that just immediately calls into `pkg/hoardcmd`.

`hoard.Node` is the main object.
Creating it with `hoard.New(...)` opens peer to peer connections, and on-disk databases.
These resources are released by calling `hoard.Node.Close()`

### User Interface
The UI is built with React.
It's all in the `ui/` directory.
To work on it you just `cd ui` and `yarn start`.
This will run a hot-reloading server on the default port `3000`.
In dev-mode API calls will be proxied to the hoard default address `localhost:6026`.
In production the server process will serve the user interface and intercept calls to the API.

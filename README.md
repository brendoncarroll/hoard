# Hoard

Hoard is a tag oriented content management system built on top of the [blobcache](https://github.com/brendoncarroll/blobcache) storage network using the file and directory structures from [WebFS](https://github.com/brendoncarroll/webfs).

Hoard aims to let normal non-technical users organize content, share files, and coordinate group archival.

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

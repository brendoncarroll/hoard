# Hoard

Hoard is a tag oriented content management system built on top of the [blobcache](https://github.com/brendoncarroll/blobcache) storage network using the file and directory structures from [WebFS](https://github.com/brendoncarroll/webfs).

Hoard aims to let normal non-technical users organize content, share files, and coordinate group archival.

## Examples
You can `go run ./cmd/hoard` to add files or serve the http API.
Hoard will create its database and initialize blobcache in the current working directory.

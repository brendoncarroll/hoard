
.PHONY: test docker

test:
	go test ./pkg/...

docker:
	docker build -t hoard:latest .
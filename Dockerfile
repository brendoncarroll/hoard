FROM node:alpine as js_builder
RUN apk add yarn
WORKDIR /app
COPY ./ui .
RUN yarn build

FROM golang:1.13 as go_builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go install -v ./cmd/hoard

FROM alpine:latest
WORKDIR /app
RUN mkdir /data && mkdir /content && mkdir /app/ui
COPY --from=js_builder /app/build /app/ui
COPY --from=go_builder /go/bin/hoard .
EXPOSE 6026
CMD ["./hoard", "--data-dir=/data", "--content-dir=/content", "--ui-dir=/app/ui", "run"]

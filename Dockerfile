FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY server/go.mod server/go.su[m] ./
RUN go mod download
COPY server/ .
RUN CGO_ENABLED=0 go build -o /tracker ./cmd/tracker

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /tracker /usr/local/bin/tracker
ENTRYPOINT ["tracker"]

FROM golang:1.26.2-alpine AS builder

WORKDIR /src/tools

COPY tools/go.mod tools/go.sum ./

RUN go mod download

WORKDIR /src
COPY . .

WORKDIR /src/tools
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/leoxy-bench ./cmd

FROM alpine:3.22

COPY --from=builder /out/leoxy-bench /usr/local/bin/leoxy-bench

ENTRYPOINT ["/usr/local/bin/leoxy-bench"]

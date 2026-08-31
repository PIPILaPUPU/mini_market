FROM golang:1.25-alpine AS builder

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go install \
    -tags="no_mysql no_clickhouse no_sqlite3 no_ydb no_libsql no_mssql no_vertica" \
    github.com/pressly/goose/v3/cmd/goose@v3.27.3

FROM alpine:3.22

RUN apk add --no-cache ca-certificates
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY auth-app/migrations /migrations

ENTRYPOINT ["goose", "-dir", "/migrations"]

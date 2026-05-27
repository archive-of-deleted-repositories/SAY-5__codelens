# Multi-stage build for the indexer and api binaries. The build context is the
# repo root so the corpus and db migrations are available to the images.
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY server/go.mod server/go.sum ./server/
WORKDIR /src/server
RUN go mod download
WORKDIR /src
COPY server ./server
RUN CGO_ENABLED=0 go -C server build -o /out/indexer ./cmd/indexer \
 && CGO_ENABLED=0 go -C server build -o /out/api ./cmd/api

FROM alpine:3.20 AS indexer
WORKDIR /app
COPY --from=build /out/indexer /usr/local/bin/indexer
COPY db ./db
COPY corpus ./corpus
ENTRYPOINT ["indexer"]

FROM alpine:3.20 AS api
WORKDIR /app
COPY --from=build /out/api /usr/local/bin/api
COPY db ./db
EXPOSE 8080
ENTRYPOINT ["api"]

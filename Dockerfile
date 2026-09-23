FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /imgo ./cmd/imgo
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/* && useradd --uid 10001 --create-home imgo
WORKDIR /app
COPY --from=build /imgo /app/imgo
COPY public /app/public
COPY data /app/data
RUN mkdir -p /app/public/storage && chown -R imgo:imgo /app/public/storage
USER imgo
ENV IMGO_ADDR=0.0.0.0:8080 PUBLIC_DIR=/app/public IP_DATABASE=/app/data/17monipdb.dat
EXPOSE 8080
ENTRYPOINT ["/app/imgo"]

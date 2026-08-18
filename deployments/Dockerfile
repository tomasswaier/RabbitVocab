# --- build stage ---
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

# --- final stage ---
FROM gcr.io/distroless/static-debian12
COPY --from=build /out/api /api
COPY --from=build /out/migrate /migrate
COPY internal/db/migrations /migrations
COPY web /web
ENTRYPOINT ["/api"]

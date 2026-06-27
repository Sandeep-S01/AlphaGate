FROM golang:1.23-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/backfill ./cmd/backfill
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/strategy ./cmd/strategy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/risk ./cmd/risk
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/execution ./cmd/execution

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S sentra \
	&& adduser -S sentra -G sentra

WORKDIR /app

COPY --from=build /out/ /app/
COPY migrations /app/migrations
COPY web /app/web

USER sentra

EXPOSE 8080

CMD ["/app/api"]

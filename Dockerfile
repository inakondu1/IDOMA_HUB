FROM golang:1.22.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o idoma-hub .

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/idoma-hub .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static
COPY --from=builder /app/verified_idoma.csv .
COPY --from=builder /app/everyday_idoma_phrases.csv .

RUN mkdir -p /app/static/uploads/posts

EXPOSE 8080

CMD ["./idoma-hub"]

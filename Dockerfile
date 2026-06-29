FROM golang:1.26.1-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nobody-is-perfect-go .

FROM scratch

WORKDIR /app

COPY --from=build /out/nobody-is-perfect-go /app/nobody-is-perfect-go
COPY words.txt /app/words.txt

ENV NIP_HOST=0.0.0.0
ENV NIP_PORT=8080
ENV NIP_API_BASE_URL=http://localhost:8080
ENV NIP_MAX_CONCURRENT_GAMES=100
ENV NIP_WORDLIST_PATH=/app/words.txt

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/app/nobody-is-perfect-go"]

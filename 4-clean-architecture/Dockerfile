# Stage 1: Build
FROM golang:alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN if [ -f "./bin/app" ]; then \
        mkdir -p /out && cp ./bin/app /out/app; \
    else \
        CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app .; \
    fi

# Stage 2: Runtime
FROM alpine:3.21

WORKDIR /app

# Buat non-root user 'app' (UID 10001) untuk keamanan sesuai Modul 12 Langkah 13
RUN addgroup -S app && adduser -S -G app -u 10001 app \
    && mkdir -p /app/logs \
    && chown -R app:app /app

# Salin binary hasil kompilasi dari stage builder
COPY --from=builder /out/app /app/app

# Jalankan container menggunakan non-root user
USER app

EXPOSE 3000

ENTRYPOINT ["./app"]
CMD ["-role=api"]

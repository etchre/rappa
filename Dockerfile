FROM golang:1.26-alpine AS build

WORKDIR /build

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/rappa .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates nodejs py3-pip python3 tzdata \
  && python3 -m pip install --break-system-packages --no-cache-dir yt-dlp ytmusicapi

WORKDIR /app
COPY --from=build /out/rappa /app/rappa
COPY ytmusic_yt_dlp_test.py /app/ytmusic_yt_dlp_test.py

CMD ["/app/rappa"]

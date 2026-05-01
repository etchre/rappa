FROM golang:1.26-alpine AS build

WORKDIR /build

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/rappa .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates py3-pip python3 tzdata \
  && pip3 install --break-system-packages --no-cache-dir yt-dlp

WORKDIR /app
COPY --from=build /out/rappa /app/rappa

CMD ["/app/rappa"]

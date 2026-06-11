# ---- build stage ----
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/wg-panel ./main.go

# ---- run stage ----
FROM alpine:3.20
WORKDIR /app
# wireguard-tools is optional (control is via netlink), iproute2 helps debugging
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/wg-panel .
EXPOSE 8080
CMD ["./wg-panel"]

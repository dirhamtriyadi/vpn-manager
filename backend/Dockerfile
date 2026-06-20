# ---- build stage ----
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/vpn-manager ./main.go

# ---- run stage ----
FROM alpine:3.20
WORKDIR /app
# ca-certificates for TLS; iptables is required for client internet (NAT/masquerade).
# wireguard-tools is optional (control is via netlink), iproute2 helps debugging.
RUN apk add --no-cache ca-certificates iptables
COPY --from=builder /app/bin/vpn-manager .
EXPOSE 8080
CMD ["./vpn-manager"]

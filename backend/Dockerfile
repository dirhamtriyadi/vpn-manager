# ---- build stage ----
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/vpn-manager ./main.go

# ---- run stage ----
# Debian (not Alpine) for the runtime so iptables uses the nftables backend by
# default, matching modern hosts (Debian/Ubuntu/Proxmox, which is what this runs
# on). Alpine's iptables defaults to the LEGACY backend: rules written from the
# container then land in the legacy tables, which are invisible to the host's
# nft `iptables -S` and bypassed by Docker's own nft FORWARD chain — silently
# breaking the port-forward DNAT and the WireGuard masquerade rules even though
# the apply "succeeded" inside the container.
FROM debian:bookworm-slim
WORKDIR /app
# ca-certificates for TLS; iptables (nft backend) is required for NAT/masquerade
# and port-forward DNAT; iproute2 helps on-host debugging.
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates iptables iproute2 \
 && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/bin/vpn-manager .
EXPOSE 8080
CMD ["./vpn-manager"]

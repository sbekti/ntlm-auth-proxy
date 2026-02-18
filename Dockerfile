# Build Stage
FROM golang:1.26 AS builder

WORKDIR /app
COPY go.mod ./
# COPY go.sum ./ # No dependencies yet, so go.sum might not exist
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ntlm_auth_proxy .

# Runtime Stage
FROM ubuntu:24.04

# Install winbind to get ntlm_auth binary
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    winbind \
    ca-certificates \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/ntlm_auth_proxy /usr/local/bin/ntlm_auth_proxy

# Default to server
ENTRYPOINT ["/usr/local/bin/ntlm_auth_proxy"]

FROM golang:1.25-bookworm AS builder

WORKDIR /src/services/agent

COPY services/agent/go.mod services/agent/go.sum ./
RUN go mod download

COPY services/agent/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/localid-agent ./cmd/localid-agent

FROM golang:1.25-bookworm AS runtime

RUN apt-get update -o Acquire::Retries=3 \
  && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends curl libpcsclite1 \
  && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/localid-agent /usr/local/bin/localid-agent

EXPOSE 17443

ENTRYPOINT ["/usr/local/bin/localid-agent"]
CMD ["--config", "/config/agent.config.json"]

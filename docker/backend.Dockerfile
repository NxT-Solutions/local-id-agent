FROM golang:1.25-bookworm AS builder

WORKDIR /src/services/agent

COPY services/agent/go.mod services/agent/go.sum ./
RUN go mod download

COPY services/agent/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mock-backend ./cmd/mock-backend

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/mock-backend /usr/local/bin/mock-backend

EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/mock-backend"]

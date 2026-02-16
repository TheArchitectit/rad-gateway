FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o rad-gateway ./cmd/rad-gateway

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl jq
WORKDIR /root/
COPY --from=builder /app/rad-gateway /usr/local/bin/
EXPOSE 8090
CMD ["/usr/local/bin/rad-gateway"]

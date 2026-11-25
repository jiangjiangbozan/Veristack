FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -o /out/veristack ./cmd/veristack

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/server /app/server
COPY --from=builder /out/veristack /app/veristack
ENV VERISTACK_STORAGE_LOCALROOT=/data
EXPOSE 8080
ENTRYPOINT ["/app/server"]

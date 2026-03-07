FROM golang:1.24-alpine AS builder

WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/. ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/data-mesh-backend ./main.go

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/data-mesh-backend /app/data-mesh-backend
EXPOSE 8080
ENTRYPOINT ["/app/data-mesh-backend"]

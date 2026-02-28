FROM golang:1.23-alpine AS builder
WORKDIR /src/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/data-pipeline-backend .

FROM alpine:3.20
RUN adduser -D -H -u 10001 appuser
USER appuser
WORKDIR /app
COPY --from=builder /out/data-pipeline-backend /app/data-pipeline-backend
ENV APP_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/data-pipeline-backend"]

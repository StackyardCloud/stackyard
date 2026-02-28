FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /health-basic ./examples/health/health-basic
ENTRYPOINT ["/health-basic"]

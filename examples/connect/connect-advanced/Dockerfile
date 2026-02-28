FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /connect-advanced ./examples/connect/connect-advanced
ENTRYPOINT ["/connect-advanced"]

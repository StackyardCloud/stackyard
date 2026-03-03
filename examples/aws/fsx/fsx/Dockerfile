FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fsx ./examples/aws/fsx/fsx
ENTRYPOINT ["/fsx"]

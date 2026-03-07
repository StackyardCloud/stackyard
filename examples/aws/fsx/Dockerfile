FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /fsx ./examples/aws/fsx
ENTRYPOINT ["/fsx"]

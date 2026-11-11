FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /fsx ./examples/aws/fsx
ENTRYPOINT ["/fsx"]

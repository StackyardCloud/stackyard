FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /fms ./examples/aws/fms/fms
ENTRYPOINT ["/fms"]

FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /networkflowmonitor ./examples/aws/networkflowmonitor
ENTRYPOINT ["/networkflowmonitor"]

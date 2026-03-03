FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /networkflowmonitor ./examples/aws/networkflowmonitor/networkflowmonitor
ENTRYPOINT ["/networkflowmonitor"]

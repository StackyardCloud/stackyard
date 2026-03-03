FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /networkflowmonitor-basic ./examples/aws/networkflowmonitor/networkflowmonitor-basic
ENTRYPOINT ["/networkflowmonitor-basic"]

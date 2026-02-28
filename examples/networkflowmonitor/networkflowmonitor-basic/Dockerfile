FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /networkflowmonitor-basic ./examples/networkflowmonitor/networkflowmonitor-basic
ENTRYPOINT ["/networkflowmonitor-basic"]

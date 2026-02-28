FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /networkflowmonitor-advanced ./examples/networkflowmonitor/networkflowmonitor-advanced
ENTRYPOINT ["/networkflowmonitor-advanced"]

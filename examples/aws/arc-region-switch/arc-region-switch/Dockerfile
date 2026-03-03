FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /arc-region-switch ./examples/aws/arc-region-switch/arc-region-switch
ENTRYPOINT ["/arc-region-switch"]

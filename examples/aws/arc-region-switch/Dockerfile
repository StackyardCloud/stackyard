FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /arc-region-switch ./examples/aws/arc-region-switch
ENTRYPOINT ["/arc-region-switch"]

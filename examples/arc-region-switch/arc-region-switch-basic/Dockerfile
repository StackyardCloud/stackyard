FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /arc-region-switch-basic ./examples/arc-region-switch/arc-region-switch-basic
ENTRYPOINT ["/arc-region-switch-basic"]

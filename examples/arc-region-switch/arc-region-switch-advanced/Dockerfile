FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /arc-region-switch-advanced ./examples/arc-region-switch/arc-region-switch-advanced
ENTRYPOINT ["/arc-region-switch-advanced"]

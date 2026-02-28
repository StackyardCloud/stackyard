FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /config-advanced ./examples/config/config-advanced
ENTRYPOINT ["/config-advanced"]

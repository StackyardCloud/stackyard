FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /keyspaces-advanced ./examples/keyspaces/keyspaces-advanced
ENTRYPOINT ["/keyspaces-advanced"]

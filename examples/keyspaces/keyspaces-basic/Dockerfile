FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /keyspaces-basic ./examples/keyspaces/keyspaces-basic
ENTRYPOINT ["/keyspaces-basic"]

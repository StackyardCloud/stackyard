FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-multitrack ./examples/aws/ivs-multitrack/ivs-multitrack
ENTRYPOINT ["/ivs-multitrack"]

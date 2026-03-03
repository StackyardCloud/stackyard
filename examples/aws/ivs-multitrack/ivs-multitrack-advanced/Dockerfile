FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-multitrack-advanced ./examples/aws/ivs-multitrack/ivs-multitrack-advanced
ENTRYPOINT ["/ivs-multitrack-advanced"]

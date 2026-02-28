FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-multitrack-basic ./examples/ivs-multitrack/ivs-multitrack-basic
ENTRYPOINT ["/ivs-multitrack-basic"]

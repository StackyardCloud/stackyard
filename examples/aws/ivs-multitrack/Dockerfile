FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-multitrack ./examples/aws/ivs-multitrack
ENTRYPOINT ["/ivs-multitrack"]

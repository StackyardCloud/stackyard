FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /ivs-multitrack ./examples/aws/ivs-multitrack
ENTRYPOINT ["/ivs-multitrack"]

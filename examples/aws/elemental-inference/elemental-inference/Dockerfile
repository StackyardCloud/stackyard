FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elemental-inference ./examples/aws/elemental-inference/elemental-inference
ENTRYPOINT ["/elemental-inference"]

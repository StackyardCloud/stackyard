FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /elemental-inference ./examples/aws/elemental-inference
ENTRYPOINT ["/elemental-inference"]

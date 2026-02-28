FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elemental-inference-advanced ./examples/elemental-inference/elemental-inference-advanced
ENTRYPOINT ["/elemental-inference-advanced"]

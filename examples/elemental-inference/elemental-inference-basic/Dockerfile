FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elemental-inference-basic ./examples/elemental-inference/elemental-inference-basic
ENTRYPOINT ["/elemental-inference-basic"]

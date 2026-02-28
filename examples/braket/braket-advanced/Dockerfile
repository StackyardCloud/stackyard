FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /braket-advanced ./examples/braket/braket-advanced
ENTRYPOINT ["/braket-advanced"]

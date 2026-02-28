FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /braket-basic ./examples/braket/braket-basic
ENTRYPOINT ["/braket-basic"]

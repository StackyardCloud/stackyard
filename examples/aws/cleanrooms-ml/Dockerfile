FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /cleanrooms-ml ./examples/aws/cleanrooms-ml
ENTRYPOINT ["/cleanrooms-ml"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cleanrooms-ml ./examples/aws/cleanrooms-ml/cleanrooms-ml
ENTRYPOINT ["/cleanrooms-ml"]

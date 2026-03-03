FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cleanrooms-ml-advanced ./examples/aws/cleanrooms-ml/cleanrooms-ml-advanced
ENTRYPOINT ["/cleanrooms-ml-advanced"]

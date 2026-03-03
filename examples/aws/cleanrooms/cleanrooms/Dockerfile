FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cleanrooms ./examples/aws/cleanrooms/cleanrooms
ENTRYPOINT ["/cleanrooms"]

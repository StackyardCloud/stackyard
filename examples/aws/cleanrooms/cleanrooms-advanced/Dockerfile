FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cleanrooms-advanced ./examples/aws/cleanrooms/cleanrooms-advanced
ENTRYPOINT ["/cleanrooms-advanced"]

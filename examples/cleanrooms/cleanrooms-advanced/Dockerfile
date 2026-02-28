FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cleanrooms-advanced ./examples/cleanrooms/cleanrooms-advanced
ENTRYPOINT ["/cleanrooms-advanced"]

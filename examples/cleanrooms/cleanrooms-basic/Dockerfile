FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cleanrooms-basic ./examples/cleanrooms/cleanrooms-basic
ENTRYPOINT ["/cleanrooms-basic"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cleanrooms-ml-basic ./examples/cleanrooms-ml/cleanrooms-ml-basic
ENTRYPOINT ["/cleanrooms-ml-basic"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptune ./examples/aws/neptune/neptune
ENTRYPOINT ["/neptune"]

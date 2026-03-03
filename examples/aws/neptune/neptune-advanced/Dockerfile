FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /neptune-advanced ./examples/aws/neptune/neptune-advanced
ENTRYPOINT ["/neptune-advanced"]

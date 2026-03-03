FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /batch-advanced ./examples/aws/batch/batch-advanced
ENTRYPOINT ["/batch-advanced"]

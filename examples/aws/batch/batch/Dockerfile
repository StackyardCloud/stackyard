FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /batch ./examples/aws/batch/batch
ENTRYPOINT ["/batch"]

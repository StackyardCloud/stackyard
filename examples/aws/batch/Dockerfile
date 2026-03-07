FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /batch ./examples/aws/batch
ENTRYPOINT ["/batch"]

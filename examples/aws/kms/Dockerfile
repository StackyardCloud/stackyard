FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /kms ./examples/aws/kms
ENTRYPOINT ["/kms"]

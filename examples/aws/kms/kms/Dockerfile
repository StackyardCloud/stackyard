FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /kms ./examples/aws/kms/kms
ENTRYPOINT ["/kms"]

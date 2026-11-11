FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /kms ./examples/aws/kms
ENTRYPOINT ["/kms"]

FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /transfer ./examples/aws/transfer
ENTRYPOINT ["/transfer"]

FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /shield ./examples/aws/shield
ENTRYPOINT ["/shield"]

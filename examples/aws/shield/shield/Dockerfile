FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /shield ./examples/aws/shield/shield
ENTRYPOINT ["/shield"]

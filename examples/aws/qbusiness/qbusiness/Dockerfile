FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /qbusiness ./examples/aws/qbusiness/qbusiness
ENTRYPOINT ["/qbusiness"]

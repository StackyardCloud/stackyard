FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /qbusiness-basic ./examples/aws/qbusiness/qbusiness-basic
ENTRYPOINT ["/qbusiness-basic"]

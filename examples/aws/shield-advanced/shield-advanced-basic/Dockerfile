FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /shield-advanced-basic ./examples/aws/shield-advanced/shield-advanced-basic
ENTRYPOINT ["/shield-advanced-basic"]


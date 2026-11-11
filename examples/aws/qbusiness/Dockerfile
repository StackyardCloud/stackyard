FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /qbusiness ./examples/aws/qbusiness
ENTRYPOINT ["/qbusiness"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /acm ./examples/aws/acm/acm
ENTRYPOINT ["/acm"]

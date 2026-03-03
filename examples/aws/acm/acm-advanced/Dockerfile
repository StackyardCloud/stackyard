FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /acm-advanced ./examples/aws/acm/acm-advanced
ENTRYPOINT ["/acm-advanced"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /support-advanced ./examples/aws/support/support-advanced
ENTRYPOINT ["/support-advanced"]

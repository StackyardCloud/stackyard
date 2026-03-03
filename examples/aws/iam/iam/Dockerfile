FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iam ./examples/aws/iam/iam
ENTRYPOINT ["/iam"]

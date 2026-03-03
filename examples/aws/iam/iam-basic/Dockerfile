FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /iam-basic ./examples/aws/iam/iam-basic
ENTRYPOINT ["/iam-basic"]

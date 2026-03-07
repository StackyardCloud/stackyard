FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /macie ./examples/aws/macie
ENTRYPOINT ["/macie"]

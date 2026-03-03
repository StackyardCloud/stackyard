FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /macie ./examples/aws/macie/macie
ENTRYPOINT ["/macie"]

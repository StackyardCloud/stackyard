FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-lake-basic ./examples/aws/security-lake/security-lake-basic
ENTRYPOINT ["/security-lake-basic"]

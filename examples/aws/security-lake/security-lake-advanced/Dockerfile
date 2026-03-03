FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-lake-advanced ./examples/aws/security-lake/security-lake-advanced
ENTRYPOINT ["/security-lake-advanced"]

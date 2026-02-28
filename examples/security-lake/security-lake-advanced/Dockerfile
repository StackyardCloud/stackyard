FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-lake-advanced ./examples/security-lake/security-lake-advanced
ENTRYPOINT ["/security-lake-advanced"]

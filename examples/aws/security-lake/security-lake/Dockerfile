FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-lake ./examples/aws/security-lake/security-lake
ENTRYPOINT ["/security-lake"]

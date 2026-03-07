FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-lake ./examples/aws/security-lake
ENTRYPOINT ["/security-lake"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-ir ./examples/aws/security-ir/security-ir
ENTRYPOINT ["/security-ir"]

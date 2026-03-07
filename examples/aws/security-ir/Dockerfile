FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-ir ./examples/aws/security-ir
ENTRYPOINT ["/security-ir"]

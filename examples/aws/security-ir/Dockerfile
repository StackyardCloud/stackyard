FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-ir ./examples/aws/security-ir
ENTRYPOINT ["/security-ir"]

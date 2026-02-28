FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-ir-advanced ./examples/security-ir/security-ir-advanced
ENTRYPOINT ["/security-ir-advanced"]

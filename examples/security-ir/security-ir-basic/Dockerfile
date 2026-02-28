FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /security-ir-basic ./examples/security-ir/security-ir-basic
ENTRYPOINT ["/security-ir-basic"]

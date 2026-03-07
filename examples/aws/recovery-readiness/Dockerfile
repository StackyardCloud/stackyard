FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /recovery-readiness ./examples/aws/recovery-readiness
ENTRYPOINT ["/recovery-readiness"]

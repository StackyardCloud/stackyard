FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recovery-readiness ./examples/aws/recovery-readiness/recovery-readiness
ENTRYPOINT ["/recovery-readiness"]

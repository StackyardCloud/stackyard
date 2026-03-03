FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recovery-readiness-advanced ./examples/aws/recovery-readiness/recovery-readiness-advanced
ENTRYPOINT ["/recovery-readiness-advanced"]

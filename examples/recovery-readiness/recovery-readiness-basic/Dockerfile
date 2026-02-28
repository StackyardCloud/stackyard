FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /recovery-readiness-basic ./examples/recovery-readiness/recovery-readiness-basic
ENTRYPOINT ["/recovery-readiness-basic"]

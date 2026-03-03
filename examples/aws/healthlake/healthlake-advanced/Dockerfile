FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /healthlake-advanced ./examples/aws/healthlake/healthlake-advanced
ENTRYPOINT ["/healthlake-advanced"]

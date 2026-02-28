FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /healthlake-advanced ./examples/healthlake/healthlake-advanced
ENTRYPOINT ["/healthlake-advanced"]

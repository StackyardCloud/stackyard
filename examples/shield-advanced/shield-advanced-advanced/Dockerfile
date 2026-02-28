FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /shield-advanced-advanced ./examples/shield-advanced/shield-advanced-advanced
ENTRYPOINT ["/shield-advanced-advanced"]


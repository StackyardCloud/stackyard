FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /qbusiness-advanced ./examples/qbusiness/qbusiness-advanced
ENTRYPOINT ["/qbusiness-advanced"]

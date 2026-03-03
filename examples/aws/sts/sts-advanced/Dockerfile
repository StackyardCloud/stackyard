FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /sts-advanced ./examples/aws/sts/sts-advanced
ENTRYPOINT ["/sts-advanced"]

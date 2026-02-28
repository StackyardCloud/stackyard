FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /cloudwatch-advanced ./examples/cloudwatch/cloudwatch-advanced
ENTRYPOINT ["/cloudwatch-advanced"]

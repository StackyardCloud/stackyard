FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /timestream-advanced ./examples/aws/timestream/timestream-advanced
ENTRYPOINT ["/timestream-advanced"]

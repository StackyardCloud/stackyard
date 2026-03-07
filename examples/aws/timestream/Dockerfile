FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /timestream ./examples/aws/timestream
ENTRYPOINT ["/timestream"]

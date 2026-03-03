FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /timestream-basic ./examples/aws/timestream/timestream-basic
ENTRYPOINT ["/timestream-basic"]

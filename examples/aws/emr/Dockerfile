FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /emr ./examples/aws/emr
ENTRYPOINT ["/emr"]

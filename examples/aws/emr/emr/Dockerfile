FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /emr ./examples/aws/emr/emr
ENTRYPOINT ["/emr"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /emr-basic ./examples/aws/emr/emr-basic
ENTRYPOINT ["/emr-basic"]

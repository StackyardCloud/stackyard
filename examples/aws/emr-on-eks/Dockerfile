FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /emr-on-eks ./examples/aws/emr-on-eks
ENTRYPOINT ["/emr-on-eks"]

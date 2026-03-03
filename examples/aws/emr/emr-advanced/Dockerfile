FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /emr-advanced ./examples/aws/emr/emr-advanced
ENTRYPOINT ["/emr-advanced"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /redshift-serverless ./examples/aws/redshift-serverless/redshift-serverless
ENTRYPOINT ["/redshift-serverless"]

FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /redshift-serverless ./examples/aws/redshift-serverless
ENTRYPOINT ["/redshift-serverless"]

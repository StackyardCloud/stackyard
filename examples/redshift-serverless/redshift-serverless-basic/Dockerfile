FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /redshift-serverless-basic ./examples/redshift-serverless/redshift-serverless-basic
ENTRYPOINT ["/redshift-serverless-basic"]

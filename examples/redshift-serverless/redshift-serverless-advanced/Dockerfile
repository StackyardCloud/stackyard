FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /redshift-serverless-advanced ./examples/redshift-serverless/redshift-serverless-advanced
ENTRYPOINT ["/redshift-serverless-advanced"]

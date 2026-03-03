FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /emr-serverless ./examples/aws/emr-serverless/emr-serverless
ENTRYPOINT ["/emr-serverless"]

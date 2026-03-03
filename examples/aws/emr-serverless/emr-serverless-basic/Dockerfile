FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /emr-serverless-basic ./examples/aws/emr-serverless/emr-serverless-basic
ENTRYPOINT ["/emr-serverless-basic"]

FROM golang:1.25 AS build
WORKDIR /src
COPY . .
RUN go build -o /emr-serverless ./examples/aws/emr-serverless
ENTRYPOINT ["/emr-serverless"]

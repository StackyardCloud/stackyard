FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /emr-serverless-advanced ./examples/emr-serverless/emr-serverless-advanced
ENTRYPOINT ["/emr-serverless-advanced"]

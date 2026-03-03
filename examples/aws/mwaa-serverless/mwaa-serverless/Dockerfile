FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mwaa-serverless ./examples/aws/mwaa-serverless/mwaa-serverless
ENTRYPOINT ["/mwaa-serverless"]

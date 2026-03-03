FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mwaa-serverless-basic ./examples/aws/mwaa-serverless/mwaa-serverless-basic
ENTRYPOINT ["/mwaa-serverless-basic"]

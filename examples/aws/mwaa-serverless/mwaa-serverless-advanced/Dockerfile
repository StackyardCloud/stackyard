FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mwaa-serverless-advanced ./examples/aws/mwaa-serverless/mwaa-serverless-advanced
ENTRYPOINT ["/mwaa-serverless-advanced"]

FROM golang:1.25 AS build
WORKDIR /src
COPY . .
RUN go build -o /mwaa-serverless ./examples/aws/mwaa-serverless
ENTRYPOINT ["/mwaa-serverless"]

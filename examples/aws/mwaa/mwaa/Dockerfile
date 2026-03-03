FROM golang:1.24 AS build
WORKDIR /src
COPY . .
RUN go build -o /mwaa ./examples/aws/mwaa/mwaa
ENTRYPOINT ["/mwaa"]

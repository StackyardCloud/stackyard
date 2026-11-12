FROM golang:1.25 AS build
WORKDIR /src
COPY . .
RUN go build -o /mwaa ./examples/aws/mwaa
ENTRYPOINT ["/mwaa"]

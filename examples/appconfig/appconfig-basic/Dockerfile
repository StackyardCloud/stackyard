FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appconfig-basic ./examples/appconfig/appconfig-basic
ENTRYPOINT ["/appconfig-basic"]

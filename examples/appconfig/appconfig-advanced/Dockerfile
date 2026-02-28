FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /appconfig-advanced ./examples/appconfig/appconfig-advanced
ENTRYPOINT ["/appconfig-advanced"]

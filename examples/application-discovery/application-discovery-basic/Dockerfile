FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /application-discovery-basic ./examples/application-discovery/application-discovery-basic
ENTRYPOINT ["/application-discovery-basic"]

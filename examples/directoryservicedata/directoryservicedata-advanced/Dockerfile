FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directoryservicedata-advanced ./examples/directoryservicedata/directoryservicedata-advanced
ENTRYPOINT ["/directoryservicedata-advanced"]

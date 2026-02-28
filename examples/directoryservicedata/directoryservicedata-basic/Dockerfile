FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directoryservicedata-basic ./examples/directoryservicedata/directoryservicedata-basic
ENTRYPOINT ["/directoryservicedata-basic"]

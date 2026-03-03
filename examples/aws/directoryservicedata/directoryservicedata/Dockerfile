FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directoryservicedata ./examples/aws/directoryservicedata/directoryservicedata
ENTRYPOINT ["/directoryservicedata"]

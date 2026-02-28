FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directconnect-basic ./examples/directconnect/directconnect-basic
ENTRYPOINT ["/directconnect-basic"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directconnect-advanced ./examples/aws/directconnect/directconnect-advanced
ENTRYPOINT ["/directconnect-advanced"]

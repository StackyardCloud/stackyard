FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /directconnect ./examples/aws/directconnect/directconnect
ENTRYPOINT ["/directconnect"]

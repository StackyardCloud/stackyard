FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /directconnect ./examples/aws/directconnect
ENTRYPOINT ["/directconnect"]

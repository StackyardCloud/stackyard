FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /nova-act ./examples/aws/nova-act/nova-act
ENTRYPOINT ["/nova-act"]

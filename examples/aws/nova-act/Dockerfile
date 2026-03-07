FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /nova-act ./examples/aws/nova-act
ENTRYPOINT ["/nova-act"]

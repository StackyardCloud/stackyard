FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /nova-act-advanced ./examples/aws/nova-act/nova-act-advanced
ENTRYPOINT ["/nova-act-advanced"]

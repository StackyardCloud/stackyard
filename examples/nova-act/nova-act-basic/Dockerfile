FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /nova-act-basic ./examples/nova-act/nova-act-basic
ENTRYPOINT ["/nova-act-basic"]

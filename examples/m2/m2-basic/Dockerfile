FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /m2-basic ./examples/m2/m2-basic
ENTRYPOINT ["/m2-basic"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /mpa-basic ./examples/mpa/mpa-basic
ENTRYPOINT ["/mpa-basic"]

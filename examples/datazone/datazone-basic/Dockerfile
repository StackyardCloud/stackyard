FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /datazone-basic ./examples/datazone/datazone-basic
CMD ["/datazone-basic"]


FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /datazone ./examples/aws/datazone/datazone
CMD ["/datazone"]


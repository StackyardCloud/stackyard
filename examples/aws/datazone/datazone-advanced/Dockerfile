FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /datazone-advanced ./examples/aws/datazone/datazone-advanced
CMD ["/datazone-advanced"]


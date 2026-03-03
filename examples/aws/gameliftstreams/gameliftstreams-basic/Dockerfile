FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /gameliftstreams-basic ./examples/aws/gameliftstreams/gameliftstreams-basic
CMD ["/gameliftstreams-basic"]

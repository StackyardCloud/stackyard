FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /gameliftstreams ./examples/aws/gameliftstreams/gameliftstreams
CMD ["/gameliftstreams"]

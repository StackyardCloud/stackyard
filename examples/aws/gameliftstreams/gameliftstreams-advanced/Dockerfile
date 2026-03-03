FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /gameliftstreams-advanced ./examples/aws/gameliftstreams/gameliftstreams-advanced
CMD ["/gameliftstreams-advanced"]

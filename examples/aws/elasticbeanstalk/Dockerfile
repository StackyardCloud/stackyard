FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticbeanstalk ./examples/aws/elasticbeanstalk
ENTRYPOINT ["/elasticbeanstalk"]

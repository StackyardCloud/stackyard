FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticbeanstalk-basic ./examples/elasticbeanstalk/elasticbeanstalk-basic
ENTRYPOINT ["/elasticbeanstalk-basic"]

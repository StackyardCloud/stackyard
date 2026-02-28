FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /elasticbeanstalk-advanced ./examples/elasticbeanstalk/elasticbeanstalk-advanced
ENTRYPOINT ["/elasticbeanstalk-advanced"]

FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /applicationsignals ./examples/aws/applicationsignals/applicationsignals
ENTRYPOINT ["/applicationsignals"]

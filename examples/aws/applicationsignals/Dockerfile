FROM golang:1.24-alpine
WORKDIR /app
COPY . .
RUN go build -o /applicationsignals ./examples/aws/applicationsignals
ENTRYPOINT ["/applicationsignals"]

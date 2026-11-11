FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /applicationsignals ./examples/aws/applicationsignals
ENTRYPOINT ["/applicationsignals"]

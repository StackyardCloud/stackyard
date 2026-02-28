FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /applicationsignals-basic ./examples/applicationsignals/applicationsignals-basic
ENTRYPOINT ["/applicationsignals-basic"]

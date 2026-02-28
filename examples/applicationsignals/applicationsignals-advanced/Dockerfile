FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o /applicationsignals-advanced ./examples/applicationsignals/applicationsignals-advanced
ENTRYPOINT ["/applicationsignals-advanced"]

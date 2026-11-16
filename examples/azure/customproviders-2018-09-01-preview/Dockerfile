FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /customproviders-2018-09-01-preview ./examples/azure/customproviders-2018-09-01-preview

ENTRYPOINT ["/customproviders-2018-09-01-preview"]

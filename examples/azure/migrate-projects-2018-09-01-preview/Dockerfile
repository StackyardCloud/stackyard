FROM golang:1.25-alpine

WORKDIR /app
COPY . .
RUN go build -o /migrate-projects-2018-09-01-preview ./examples/azure/migrate-projects-2018-09-01-preview

ENTRYPOINT ["/migrate-projects-2018-09-01-preview"]

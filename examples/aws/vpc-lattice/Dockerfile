FROM golang:1.25-alpine
WORKDIR /app
COPY . .
RUN go build -o /vpc-lattice ./examples/aws/vpc-lattice
ENTRYPOINT ["/vpc-lattice"]

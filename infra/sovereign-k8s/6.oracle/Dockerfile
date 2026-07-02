FROM golang:1.25.9-alpine AS builder
WORKDIR /app
COPY go.work ./
COPY oracle ./oracle
COPY chain ./chain
RUN cd oracle && go build -o /app/bin/oracle .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/bin/oracle /usr/local/bin/oracle
EXPOSE 9200
ENTRYPOINT ["/usr/local/bin/oracle"]

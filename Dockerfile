FROM golang:1.18-buster

WORKDIR /app

RUN go version
ENV GOPATH=/

COPY . .

# install psql
RUN apt-get update
RUN apt-get -y install postgresql-client

# make wait-for-postgres.sh executable
RUN chmod +x wait-for-postgres.sh

RUN go mod download
RUN go build -o secretSanta ./cmd/bot/main.go

CMD ["./secretSanta"]
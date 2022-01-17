FROM golang:1.17

RUN go version
ENV GOPATH=/

COPY . .

# install psql
RUN apt-get update
RUN apt-get -y install postgresql-client

# make wait for postgres executable
RUN chmod +x wait-for-postgres.sh

# prepare go modules
RUN go mod tidy -compat=1.17
RUN go mod vendor

RUN go build -o ./bin/main_bot ./cmd/the_open_art_ton_bot/main.go

CMD ["./bin/main_bot"]
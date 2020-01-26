From golang:latest
RUN apt-get update
WORKDIR /go/src/app
RUN go get -u github.com/kyoh86/richgo
COPY . .

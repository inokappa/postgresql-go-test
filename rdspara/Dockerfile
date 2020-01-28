###############################
# Builder container
###############################

From golang:latest as builder
RUN apt-get update
WORKDIR /go/src/app
COPY . .

# Compile
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o rdspara .

###############################
# Exec container
###############################

From alpine:latest
COPY --from=builder /go/src/app/rdspara /rdspara
RUN apk add libc6-compat
CMD ["/rdspara"]

# To run properly, the directory containing the proto files Truss will read to
# generate a service must be mounted into the container, so that the output
# generated by Truss escapes the container.

# Using scratch as a base image would result in a slightly smaller image,
# which would be fine if Truss were a self-contained executable, but protoc
# has difficulty running on a scratch image. Using Alpine Linux alleviates
# these issues and only increases image size by about 5 MB.
FROM golang:alpine3.5
MAINTAINER lab@tune.com

RUN apk update && apk upgrade && apk add --no-cache protobuf git

RUN go version && go get -u -v github.com/gogo/protobuf/protoc-gen-go

COPY ./ $GOPATH/src/github.com/metaverse/truss

RUN go install -v github.com/metaverse/truss/...

WORKDIR  /go/src/protos

ENTRYPOINT ["truss"]

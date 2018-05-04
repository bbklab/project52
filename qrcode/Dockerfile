FROM golang:1.8.1
RUN apt -y update 
RUN apt -y install libzbar-dev zbar-tools
RUN go get -u github.com/PeterCxy/gozbar
RUN go get -u github.com/lvzhihao/gozbar
RUN go get -u github.com/makiuchi-d/gozxing/qrcode
RUN go get -u github.com/skip2/go-qrcode
RUN go get -u github.com/urfave/cli
WORKDIR /build

FROM golang:1.23.2-bookworm

ARG OPENCV_VERSION=4.10.0
ARG TMP_DIR=/tmp

RUN apt update && apt install -y git build-essential tesseract-ocr libtesseract-dev  tesseract-ocr-rus tesseract-ocr-eng sudo
RUN apt install -y unzip wget build-essential cmake curl git libgtk2.0-dev pkg-config libavcodec-dev libavformat-dev libswscale-dev libtbb-dev libjpeg-dev libpng-dev libtiff-dev libharfbuzz-dev libfreetype6-dev

WORKDIR /app/opencv
RUN curl -Lo opencv.zip https://github.com/opencv/opencv/archive/refs/tags/$OPENCV_VERSION.zip
RUN unzip -q opencv.zip
RUN curl -Lo opencv_contrib.zip https://github.com/opencv/opencv_contrib/archive/refs/tags/$OPENCV_VERSION.zip
RUN unzip -q opencv_contrib.zip

WORKDIR /app/opencv/build
RUN cmake -D CMAKE_BUILD_TYPE=RELEASE -D CMAKE_INSTALL_PREFIX=/usr/local -D BUILD_SHARED_LIBS=ON -D OPENCV_EXTRA_MODULES_PATH=/app/opencv/opencv_contrib-$OPENCV_VERSION/modules -D BUILD_DOCS=OFF -D BUILD_EXAMPLES=OFF -D BUILD_TESTS=OFF -D BUILD_PERF_TESTS=ON -D BUILD_opencv_java=NO -D BUILD_opencv_python=NO -D BUILD_opencv_python2=NO -D BUILD_opencv_python3=NO -D WITH_JASPER=OFF -D WITH_TBB=OFF -DOPENCV_GENERATE_PKGCONFIG=ON /app/opencv/opencv-$OPENCV_VERSION 
RUN make -j $(nproc --all --ignore 1)

RUN make preinstall
RUN make install

WORKDIR /app/src
COPY src .

RUN go generate api/api.go
RUN go build -o ../ocr-service . 

WORKDIR /app
RUN rm -rf src
RUN rm -rf opencv

ENTRYPOINT [ "/app/ocr-service" ]

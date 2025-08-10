FROM frolvlad/alpine-glibc:alpine-3_glibc-2.34

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

WORKDIR /opt/cl-query
COPY ./build/bin/* ./
COPY ./etc ./etc
COPY ./version ./
COPY ./docs ./docs

VOLUME /opt/cl-query/etc
VOLUME /opt/cl-query/logs

EXPOSE 8765

CMD ./querysvc

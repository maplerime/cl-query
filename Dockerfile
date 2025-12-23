FROM frolvlad/alpine-glibc:alpine-3_glibc-2.34

ARG USER_NAME=clquery
ARG USER_UID=10000
ARG USER_GID=10000

RUN addgroup -g ${USER_GID} -S ${USER_NAME} && \
    adduser -u ${USER_UID} -S ${USER_NAME} -G ${USER_NAME}

ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

WORKDIR /opt/cl-query
COPY --chown=${USER_UID}:${USER_GID} ./build/bin/* ./
COPY --chown=${USER_UID}:${USER_GID} ./etc ./etc
COPY --chown=${USER_UID}:${USER_GID} ./version ./
COPY --chown=${USER_UID}:${USER_GID} ./docs ./docs

VOLUME /opt/cl-query/etc
VOLUME /opt/cl-query/logs

USER ${USER_NAME}

EXPOSE 8765

CMD ./querysvc

# FROM scratch
FROM fedora

LABEL org.opencontainers.image.authors="smitram@gmail.com"

RUN yum install -y procps

ADD ./bin     /reaper/bin
ADD testpid1  /reaper/testpid1

CMD ["/reaper/testpid1"]

# FROM scratch
FROM fedora

LABEL org.opencontainers.image.authors="smitram@gmail.com"

RUN yum install -y procps

COPY  ./bin            /reaper/bin
COPY  testpid1         /reaper/testpid1
COPY  fixtures/config  /reaper/config

#  Use entrypoint, so we can pass command line parameters for running
#  the different tests.
ENTRYPOINT ["/reaper/bin/init.sh"]
CMD [""]

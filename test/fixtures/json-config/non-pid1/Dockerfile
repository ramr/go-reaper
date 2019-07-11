# FROM scratch
FROM fedora

MAINTAINER  smitram@gmail.com

COPY ./bin        /reaper/bin
COPY testpid1     /reaper/testpid1
COPY reaper.json  /reaper/config/reaper.json

CMD bash -c "/reaper/testpid1"

#Dockerfile for agent

FROM reg-bj.eng.vmware.com:5000/jpetazzo/dind:latest

ADD ./agent /usr/local/bin/

RUN chmod +x /usr/local/bin/agent

ADD ./run.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/run.sh

VOLUME /log/

CMD run.sh

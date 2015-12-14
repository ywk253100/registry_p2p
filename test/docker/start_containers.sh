#!/bin/bash

num=$1

for (( i=1; i<=num; i++)); do
        port=$((8000+i))
        docker run -d --privileged --net="host" \
		-e image=$2 \
		-e DOCKER_DAEMON_ARGS="--insecure-registry=10.110.187.0/23" \
		-e agent_port=$port \
		-v /root/registry_p2p/agent/:/usr/local/bin/ \
		-v /tmp/log/:/log/ \
		reg-bj.eng.vmware.com/base/dind:1.0 wrapdocker docker_test
done
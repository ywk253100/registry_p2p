#!/bin/bash

num=$1

for (( i=1; i<=num; i++)); do
        port=$((8000+i))
        docker run -d --privileged --net="host" \
		-e agent_port=$port \
		-v /root/registry_p2p/agent/:/usr/local/bin/ \
		-v /tmp/log/:/log/ \
		reg-bj.eng.vmware.com/base/dind:1.0 wrapdocker agent.sh
done
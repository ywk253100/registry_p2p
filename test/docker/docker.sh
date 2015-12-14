#!/bin/bash

ipaddr=$(ifconfig eth0 | awk '/inet addr/{print substr($2,6)}')
logFileName=${ipaddr}_${agent_port}.log

echo "" > /log/$logFileName

file="/var/run/docker.sock"
while true; do
        if [ -S "$file" ]
        then
                echo "docker is running">>/log/$logFileName
                break
        else
                echo "waiting for docker">>/log/$logFileName
                sleep 1
        fi
done

start=`date +%s`
docker pull $image>>/log/$logFileName 2>&1
status=$?
end=`date +%s`

if [ "$status" == 0 ]
then
        echo "[statistics_success] "$start" "$end" "$((end-start))>>/log/$logFileName
else
		echo "[statistics_fail]">>/log/$logFileName
fi

#/bin/bash

docker rm -f -v $(docker ps -a -q)
kill $(ps -ef | awk '/ctorrent/{print $2}')
rm /tmp/log/*
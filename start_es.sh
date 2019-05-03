#!/usr/bin/env bash

function wait_for_es {
    while true; do
        curl -s http://localhost:9200/_cluster/health | grep -q green

        if [ $? -eq 0 ]; then
            break
        fi

        echo "Waiting for elastic search"
        sleep 2
    done
}

docker create --rm --name=es -p 9200:9200 -e "xpack.security.audit.enabled=true" -e "xpack.security.audit.outputs=logfile" docker.elastic.co/elasticsearch/elasticsearch:6.6.2
docker start es

wait_for_es

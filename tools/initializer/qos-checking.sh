#!/usr/bin/env bash

ready=0
while [ $ready -eq 0 ]; do
    while read line;
    do
      if echo "$line" | grep -q "gocrane.io/cpu-qos"; then
        echo "found annotations" $line
        ready=1
        break
      fi

    done < /etc/podinfo/annotations

    if [[ $ready -eq 0 ]]
    then
        sleep 5
    fi
done

echo "QOS Initialized"
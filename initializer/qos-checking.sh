#!/usr/bin/env bash

ready=0
while [ $ready -eq 0 ]; do
    while read line;
    do
      echo $line;
      if [[ $line =~ "gocrane.io/cpu-qos" ]]
      then
        echo "into if"
        ready=1
        break
      fi
    done < ./podinfo

    if [[ $ready -eq 0 ]]
    then
        sleep 5
    fi
done

echo "QOS Initialized"
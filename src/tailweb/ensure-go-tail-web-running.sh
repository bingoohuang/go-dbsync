#!/bin/bash

while :
do
    ps auxw | grep go-tail-web.linux.bin | grep -v grep > /dev/null
    if [ $? != 0 ]
    then
        nohup ./go-tail-web.linux.bin -contextPath=/et -log=yoga:/home/yogaapp/tomcat/yoga-system/logs/catalina.out,et:/home/yogaapp/et-server/et-server.log > go-tail-web.out 2>&1 &
    fi

    sleep 10
done
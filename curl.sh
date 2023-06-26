#!/bin/bash
host=`curl http://169.254.169.254'
#status=`curl -o /dev/null -s -w %{http_code}  -H 'content-type: application/json' -X GET -d '{"serialNumber":"123"}'  https://www.gds.cc`
status=`curl -o /dev/null -s -w %{http_code} -i --connect-timeout 10 -m 10 -H "Content-Type: application/json" -X POST -d '{"username":"u","password":"1111","lang":"zh-CN"}' "http://$host:8080/app/check_mfa_info"`


echo $status

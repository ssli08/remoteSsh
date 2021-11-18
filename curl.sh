#!/bin/bash
host=`curl http://169.254.169.254'
#status=`curl -o /dev/null -s -w %{http_code}  -H 'content-type: application/json' -X GET -d '{"serialNumber":"123"}'  https://www.gdms.cloud`
status=`curl -o /dev/null -s -w %{http_code} -i --connect-timeout 10 -m 10 -H "Content-Type: application/json" -X POST -d '{"username":"yxxu","password":"9cf7bc42f97a23dfe920e48b44326566837c64f6b970b8acaeee78cfd844adc2","lang":"zh-CN"}' "http://$host:8080/app/check_mfa_info"`


echo $status

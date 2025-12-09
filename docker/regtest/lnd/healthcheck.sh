#!/bin/sh

STATUS_CODE="$(curl -ks -o /dev/null -w ''%{http_code}'' https://localhost:10009)"

[ $STATUS_CODE == "415" ] || exit 1

exit 0

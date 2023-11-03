#!/usr/bin/env bash

set -eux
cd `dirname $0`

################################################################################
echo "# Prepare"
################################################################################

# ====== env ======
cat > /tmp/prepared_env <<EOF
prepared_time="`date +'%Y-%m-%d %H:%M:%S'`"
app_log="/home/isucon/log/app/app.log"
app_journal_log="/home/isucon/log/app/journal.log"
nginx_access_log="/home/isucon/log/nginx/access.log"
nginx_error_log="/home/isucon/log/nginx/error.log"
nginx_journal_log="/home/isucon/log/nginx/journal.log"
mysql_slow_log="/var/log/mysql/mysqld-slow.log"
mysql_error_log="/var/log/mysql/error.log"
result_dir="/home/isucon/result"
EOF

# read env
# 計測用自作env
. /tmp/prepared_env

# isucon serviceで使うenv
. ./env.sh

# ====== go ======
(
	cd /home/isucon/isuumo/webapp/go
	make all
)
mkdir -p /home/isucon/log/app
#sudo logrotate -f /home/isucon/etc/logrotate.d/app
sudo systemctl restart isuumo.go.service

now=`date +'%Y%m%d%H%M%S'`

# ====== nginx ======
# mkdir -p /home/isucon/log/nginx
# sudo touch ${nginx_access_log} ${nginx_error_log}
# sudo cp ${nginx_access_log} ${nginx_access_log}.$now
# sudo truncate -s 0 ${nginx_access_log}
# sudo ls -1 ${nginx_access_log}.* | sort -r | uniq | sed -n '6,$p' | xargs rm -f
# sudo cp ${nginx_error_log} ${nginx_error_log}.$now
# sudo truncate -s 0 ${nginx_error_log}
# sudo ls -1 ${nginx_error_log}.* | sort -r | uniq | sed -n '6,$p' | xargs rm -f
# sudo nginx -t
# sudo systemctl restart nginx

# ====== openresty =====
mkdir -p /home/isucon/log/nginx
sudo touch ${nginx_access_log} ${nginx_error_log}
sudo cp ${nginx_access_log} ${nginx_access_log}.$now
sudo truncate -s 0 ${nginx_access_log}
sudo ls -1 ${nginx_access_log}.* | sort -r | uniq | sed -n '6,$p' | xargs rm -f
sudo cp ${nginx_error_log} ${nginx_error_log}.$now
sudo truncate -s 0 ${nginx_error_log}
sudo ls -1 ${nginx_error_log}.* | sort -r | uniq | sed -n '6,$p' | xargs rm -f
sudo rm -rf /var/cache/nginx/*
sudo openresty -c /home/isucon/etc/openresty/nginx.conf -t
sudo systemctl restart openresty

# ====== mysql ======
# sudo touch ${mysql_slow_log} ${mysql_error_log}
# sudo chown mysql:mysql ${mysql_slow_log} ${mysql_error_log}
# sudo cp ${mysql_slow_log} ${mysql_slow_log}.prev
# sudo truncate -s 0 ${mysql_slow_log}
# sudo cp ${mysql_error_log} ${mysql_error_log}.prev
# sudo truncate -s 0 ${mysql_error_log}
# sudo systemctl restart mysql

# ====== redis ======
# sudo systemctl restart redis-server

# slow log
# MYSQL="mysql -h${DB_HOST} -P${DB_PORT} -u${DB_USER} -p${DB_PASS} ${DB_DATABASE}"
# ${MYSQL} -e "set global slow_query_log_file = '${mysql_slow_log}'; set global long_query_time = 0; set global slow_query_log = ON;"

echo "OK"

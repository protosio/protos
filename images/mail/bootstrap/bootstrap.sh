#!/bin/bash
BASE_DIR=/tmp/bootstrap/

# INSTALL PKG's
cp $BASE_DIR/network /etc/sysconfig/network
yum makecache
yum install postfix dovecot dovecot-mysql system-switch-mail mysql-server -y
# MYSQL
service mysqld start
usr/bin/mysqladmin -u root password 'changeme'
mysql -uroot -p'changeme' < $BASE_DIR/sql/mail.sql
# POSTFIX
cp $BASE_DIR/postfix/mysql* /etc/postfix/
chown root:postfix /etc/postfix/mysql-virtual_*.cf
chmod 644 /etc/postfix/mysql-virtual_*.cf
groupadd -g 5000 vmail
useradd -g vmail -u 5000 vmail -d /home/vmail -m
/bin/bash $BASE_DIR/postfix/postfix_conf.sh
# DOVECOT
cp $BASE_DIR/dovecot/dovecot.conf /etc/dovecot/dovecot.conf
cp $BASE_DIR/dovecot/dovecot-sql.conf /etc/dovecot/dovecot-sql.conf
chgrp dovecot /etc/dovecot/dovecot-sql.conf
chmod o= /etc/dovecot/dovecot-sql.conf
# CREATE EMAILS
cp $BASE_DIR/postfix/aliases /etc/aliases
chown root:root /etc/aliases
newaliases
mysql -uroot -p'changeme' < $BASE_DIR/sql/setmailbox.sql 
# Clean
service mysqld stop

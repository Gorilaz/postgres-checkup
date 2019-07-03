#!/bin/bash
PG_VER=$1

wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
echo "deb http://apt.postgresql.org/pub/repos/apt/ xenial-pgdg main $PG_SERVER_VERSION" > /etc/apt/sources.list.d/pgdg.list
apt-get update
apt-get -y upgrade
apt-get -y install postgresql-${PG_VER} postgresql-contrib-${PG_VER} postgresql-client-${PG_VER} postgresql-server-dev-${PG_VER} && apt-get install -y postgresql-${PG_VER}-pg-stat-kcache
psql --version
echo "export PATH=\$PATH:/usr/lib/go-1.9/bin" >> ~/.profile
source ~/.profile
echo "127.0.0.2 postgres.test1.node" >> /etc/hosts # replica 1
echo "127.0.0.3 postgres.test2.node" >> /etc/hosts # replica 2
echo "127.0.0.4 postgres.test3.node" >> /etc/hosts # master

# Configure postgres

## Configure pg_hba.conf
echo "local   all all trust" > /etc/postgresql/${PG_VER}/main/pg_hba.conf
echo "host all  all    0.0.0.0/0  md5" >> /etc/postgresql/${PG_VER}/main/pg_hba.conf
echo "host all  all    ::1/128  trust" >> /etc/postgresql/${PG_VER}/main/pg_hba.conf
echo "host replication  replication    ::1/128  md5" >> /etc/postgresql/${PG_VER}/main/pg_hba.conf

# Configure postgres master node

## Configure general params
echo "listen_addresses='*'" >> /etc/postgresql/${PG_VER}/main/postgresql.conf
echo "log_filename='postgresql-${PG_VER}-main.log'" >> /etc/postgresql/${PG_VER}/main/postgresql.conf
echo "shared_preload_libraries = 'pg_stat_statements,auto_explain,pg_stat_kcache'" >> /etc/postgresql/${PG_VER}/main/postgresql.conf

## Configure general master params
echo "wal_level = hot_standby" >> /etc/postgresql/${PG_VER}/main/postgresql.conf
echo "max_wal_senders = 5" >> /etc/postgresql/${PG_VER}/main/postgresql.conf
echo "wal_keep_segments = 32" >> /etc/postgresql/${PG_VER}/main/postgresql.conf
echo "archive_mode    = on" >> /etc/postgresql/${PG_VER}/main/postgresql.conf
echo "archive_command = 'cp %p /path_to/archive/%f'" >> /etc/postgresql/${PG_VER}/main/postgresql.conf

## Start master node
/etc/init.d/postgresql start 
psql -U postgres -c "create role replication with replication password 'rEpLpAssw' login"
psql -U postgres -c 'create database dbname;'
psql -U postgres dbname -b -c 'create extension if not exists pg_stat_statements;'
psql -U postgres dbname -b -c 'create extension if not exists pg_stat_kcache;'
psql -U postgres dbname -c "create role pgusername superuser login;"
psql -U postgres -c 'show data_directory;'


function addReplica() {
  local num="$1"
  local port="$2"

  ## Configure data storage
  sudo -u postgres mkdir /var/lib/postgresql/${PG_VER}/data${num} && sudo -u postgres chmod 0700 /var/lib/postgresql/${PG_VER}/data${num}
  sudo -u postgres /usr/lib/postgresql/${PG_VER}/bin/initdb /var/lib/postgresql/${PG_VER}/data${num}
  sudo -u postgres cp /etc/postgresql/${PG_VER}/main/pg_hba.conf /var/lib/postgresql/${PG_VER}/data${num}/

  ## Configure general postgres settings
  echo "port = ${port}" >> /var/lib/postgresql/${PG_VER}/data${num}/postgresql.conf
  echo "listen_addresses='*'" >> /var/lib/postgresql/${PG_VER}/data${num}/postgresql.conf
  echo "shared_preload_libraries = 'pg_stat_statements,auto_explain,pg_stat_kcache'" >> /var/lib/postgresql/${PG_VER}/data${num}/postgresql.conf
  sudo -u postgres /usr/lib/postgresql/${PG_VER}/bin/pg_ctl -D /var/lib/postgresql/${PG_VER}/data${num} -l /var/log/postgresql/replica${num}.log start || cat /var/log/postgresql/replica${num}.log
  psql -U postgres -p ${port} -c 'show data_directory;'
  psql -U postgres -p ${port} -c 'create database dbname;'
  psql -U postgres -p ${port} dbname -b -c 'create extension if not exists pg_stat_statements;'
  psql -U postgres -p ${port} dbname -b -c 'create extension if not exists pg_stat_kcache;'
  psql -U postgres -p ${port} dbname -c "create role pgusername superuser login;"
  sudo -u postgres /usr/lib/postgresql/${PG_VER}/bin/pg_ctl -D /var/lib/postgresql/${PG_VER}/data${num} -l /var/log/postgresql/replica${num}.log stop

  ## Configure replica postgres settings
  echo "hot_standby = on" >> /var/lib/postgresql/${PG_VER}/data${num}/postgresql.conf
  echo "standby_mode = 'on'" > /var/lib/postgresql/${PG_VER}/data${num}/recovery.conf
  echo "primary_conninfo = 'host=127.0.0.4 port=5432 user=replication password=rEpLpAssw'" >> /var/lib/postgresql/${PG_VER}/data${num}/recovery.conf
  echo "trigger_file = '/var/lib/postgresql/${PG_VER}/data${num}/trigger'" >> /var/lib/postgresql/${PG_VER}/data${num}/recovery.conf
  echo "restore_command = 'cp /path_to/archive/%f "%p"'" >> /var/lib/postgresql/${PG_VER}/data${num}/recovery.conf

  ## Start replica
  sudo -u postgres /usr/lib/postgresql/${PG_VER}/bin/pg_ctl -D /var/lib/postgresql/${PG_VER}/data${num} -l /var/log/postgresql/secondary1.log start || cat /var/log/postgresql/replica${num}.log
}

addReplica 1 5433
#addReplica 2 5434

ps ax | grep postgres
psql -U postgres -p 5432 -c 'show data_directory;'
psql -U postgres -p 5433 -c 'show data_directory;'
#psql -U postgres -p 5434 -c 'show data_directory;'
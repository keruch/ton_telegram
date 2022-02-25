#!/bin/bash

start() {
	if nc -zv localhost 5432 2>/dev/null; then
		return
	fi
	SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
	cd "$SCRIPTPATH" || exit 1
	rm -rf data/kafka data/zookeeper
	mkdir -p data/postgres || exit 1
	docker compose up -d || exit 1
	echo -n waiting for readiness
	while : ; do
		if docker compose exec postgres pg_isready -U postgres >/dev/null 2>&1; then
			break
		fi
		echo -n .
		sleep 0.1
	done
	echo done
}

stop() {
	SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
	cd "$SCRIPTPATH" || exit 1
	docker compose down || exit 1
}

restart() {
	stop || exit 1
	start || exit 1
}


cmd=$1

case $cmd in
	start|stop|restart)
		$cmd
		;;
	*)
		echo "usage: $0 start|stop|restart"
		;;
esac

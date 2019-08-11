#!/bin/bash

cd "$(dirname "$0")/.." || { echo "FATAL: can't cd to script's upper path!"; exit 1; }

[[ -z "$CMD" ]] && { echo "FATAL: CMD not specified!"; exit 1; }
[[ -z "$OUTFILE" ]] && OUTFILE=logs/stdout.log
[[ -z "$PIDFILE" ]] && PIDFILE=logs/run.pid

set -uo pipefail

mkdir -p "$(dirname "$OUTFILE")"
mkdir -p "$(dirname "$PIDFILE")"

running() {
    test -f "$PIDFILE" && xargs kill -0 < "$PIDFILE" &> /dev/null
}

status() {
    running && {
        echo "running"
        return 0
    }

    echo "stopped"
    return 1
}

start() {
    running && {
        echo "ERROR: is already running!"
        return 1
    }

    (
    set -m
    sh -c "while true; do echo -e \"\n--------Run at \$(date -Iseconds)\" &>> $OUTFILE; $CMD &>> $OUTFILE; sleep 1; done" &
    echo $! > "$PIDFILE" || exit 1
    )

    echo "started"
}

stop() {
    test -f "$PIDFILE" && {
        xargs -I{} kill -15 -{} < "$PIDFILE" && rm -f "$PIDFILE"
        [[ $? == 0 ]] || {
            echo "WARNING: stop failed, you should check the pid file $(readlink -f "$PIDFILE")"
            return 0
        }
    }

    echo "stopped"
    return 0
}

force_stop() {
    test -f "$PIDFILE" && {
        xargs -I{} kill -9 -{} < "$PIDFILE" && rm -f "$PIDFILE"
        [[ $? == 0 ]] || {
            echo "ERROR: kill failed, you should check the pid file $(readlink -f "$PIDFILE")"
            return 1
        }
    }

    echo "killed"
    return 0
}

[[ $# -le 0 ]] && { status; exit; }

case "${1}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    force_stop)
        force_stop
        ;;
    restart)
        stop && sleep 1 && start
        ;;
    status)
        status
        ;;
    *)
        echo "${0} <start|stop|restart|force_stop|status>"
        exit 1
        ;;
esac

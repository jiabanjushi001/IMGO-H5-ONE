#!/bin/sh
# Lightweight supervisor; use systemd for automatic startup after server reboot.
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SCRIPT="$ROOT/run.sh"
STATE="$ROOT/run"
PIDFILE="$STATE/supervisor.pid"
LOCK="$STATE/supervisor.lock"
LOG="$ROOT/logs/imgo.log"
mkdir -p "$STATE" "$ROOT/logs"
umask 077
ensure_executable() {
 [ -f "$ROOT/bin/imgo" ] || { echo 'Missing file bin/imgo; extract the complete Linux server package.' >&2; return 1; }
 if [ ! -x "$ROOT/bin/imgo" ]; then
  chmod 755 "$ROOT/bin/imgo" 2>/dev/null || true
 fi
 [ -x "$ROOT/bin/imgo" ] || { echo 'bin/imgo exists but is not executable; run: chmod 755 bin/imgo' >&2; return 1; }
}
running() {
 [ -f "$PIDFILE" ] || return 1
 PID=$(cat "$PIDFILE")
 case "$PID" in ''|*[!0-9]*) return 1;; esac
 kill -0 "$PID" 2>/dev/null || return 1
 ps -p "$PID" -o args= 2>/dev/null | grep -F -- "$SCRIPT __supervise" >/dev/null
}
case "${1:-start}" in
 __supervise)
  ensure_executable || exit 1
  echo "$$" > "$PIDFILE"
  CHILD=''
  cleanup() {
   trap '' TERM INT
   if [ -n "$CHILD" ]; then
    kill -TERM "$CHILD" 2>/dev/null || true
    wait "$CHILD" 2>/dev/null || true
   fi
   rm -f "$PIDFILE"
   rmdir "$LOCK" 2>/dev/null || true
  }
  trap 'exit 0' TERM INT
  trap cleanup EXIT
  cd "$ROOT"
  set -a
  # Normalize Windows CRLF in memory; leave the original .env unchanged.
eval "$(sed "s/$(printf '\r')$//" ./.env)"
  set +a
  while :; do
   echo "$(date '+%F %T') starting Imgo"
   ./bin/imgo &
   CHILD=$!
   CODE=0
   wait "$CHILD" || CODE=$?
   CHILD=''
   echo "$(date '+%F %T') Imgo exited ($CODE); restarting in 3 seconds"
   sleep 3 &
   CHILD=$!
   wait "$CHILD" || true
   CHILD=''
  done
  ;;
 start)
  if running; then echo "Imgo supervisor is running (PID $PID)"; exit 0; fi
  [ -f "$ROOT/.env" ] || { echo 'Missing .env: copy and configure .env.example first.' >&2; exit 1; }
  ensure_executable || exit 1
  if ! mkdir "$LOCK" 2>/dev/null; then
   echo 'Supervisor lock exists. Check run/supervisor.pid before removing a stale lock.' >&2
   exit 1
  fi
  nohup /bin/sh "$SCRIPT" __supervise >> "$LOG" 2>&1 < /dev/null &
  COUNT=0
  while [ "$COUNT" -lt 5 ]; do
   if running; then echo "Imgo supervisor started (PID $PID); log: $LOG"; exit 0; fi
   sleep 1
   COUNT=$((COUNT + 1))
  done
  echo "Supervisor failed to start; inspect $LOG" >&2
  exit 1
  ;;
 stop)
  if ! running; then echo 'Imgo supervisor is not running'; exit 0; fi
  kill -TERM "$PID"
  COUNT=0
  while running; do
   COUNT=$((COUNT + 1))
   [ "$COUNT" -lt 30 ] || { echo 'Stop timed out; inspect the process before retrying.' >&2; exit 1; }
   sleep 1
  done
  echo 'Imgo stopped'
  ;;
 restart)
  /bin/sh "$SCRIPT" stop
  exec /bin/sh "$SCRIPT" start
  ;;
 status)
  if running; then
   echo "Supervisor running (PID $PID)"
   ps -o pid=,ppid=,args= -ax | awk -v parent="$PID" '$2 == parent'
  else echo 'Imgo supervisor is not running'; exit 1; fi
  ;;
 logs) tail -n 100 -f "$LOG" ;;
 *) echo "Usage: $0 {start|stop|restart|status|logs}"; exit 1 ;;
esac

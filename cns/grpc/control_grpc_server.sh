#!/bin/bash

SERVICE_DIR="../service"
GRPC_SERVER_LOG="grpc_server.log"
PID_FILE="grpc_server.pid"

start_server() {
  echo "Starting gRPC server..."
  cd $SERVICE_DIR
  nohup go run . > "../grpc/$GRPC_SERVER_LOG" 2>&1 &
  echo $! > "../grpc/$PID_FILE"
  cd - > /dev/null
  echo "gRPC server started with PID $(cat ../grpc/$PID_FILE)"
}

stop_server() {
  # Killing processes listening on ports 8080, 9090, and 10090
  for port in 8080 9090 10090; do
    pid=$(lsof -ti tcp:$port)
    if [ -n "$pid" ]; then
      echo "Stopping process listening on port $port with PID $pid..."
      kill -9 $pid
      echo "Process on port $port stopped."
    else
      echo "No process found listening on port $port."
    fi
  done

  # Also stop any leftover server process using the PID file
  if [ -f "../grpc/$PID_FILE" ]; then
    echo "Stopping gRPC server with PID $(cat ../grpc/$PID_FILE)..."
    kill $(cat "../grpc/$PID_FILE")
    rm "../grpc/$PID_FILE"
    echo "gRPC server stopped."
  else
    echo "No PID file found. Server might not be running."
  fi
}

restart_server() {
  stop_server
  sleep 2
  start_server
}

case $1 in
  start)
    start_server
    ;;
  stop)
    stop_server
    ;;
  restart)
    restart_server
    ;;
  *)
    echo "Usage: $0 {start|stop|restart}"
    ;;
esac

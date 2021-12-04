#!/bin/bash

# This script will start-off the demo application: 
# 
# . A cluster with 3 nodes on ports (raft, http): (9999, 9000) (9998, 9001) (9997, 9002).
# . 2 instances of the dummy (auth) applications on ports (8080, 8081) 
# . 3 instances of the dummy (web)  applications on ports (8082, 8083)

CWD=$(pwd)
cd $CWD/server
go run main.go --port=9000 --rport=9999 --sdir="/tmp/heartbeat/node-1" & 

# Wait for the leader to start
sleep 5
go run main.go --leader="127.0.0.1:9000" --port=9001 --rport=9998 --sdir="/tmp/heartbeat/node-2" & 
go run main.go --leader="127.0.0.1:9000" --port=9002 --rport=9997 --sdir="/tmp/heartbeat/node-3" & 


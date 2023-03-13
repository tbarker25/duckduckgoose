# duckduckgoose
A system that selects a single node as a “Goose” within a cluster of nodes. Every other node should be considered a “Duck”.

## Usage

```sh
# Build the binary
go build

# Spin up some nodes locally (recommend either 3 or 5)
./duckduckgoose -state_dir /tmp/0 -raft_addr=:8080 -api_addr=:9090 -bootstrap_cluster
./duckduckgoose -state_dir /tmp/1 -raft_addr=:8081 -api_addr=:9091 -join=localhost:8080
./duckduckgoose -state_dir /tmp/2 -raft_addr=:8082 -api_addr=:9092 -join=localhost:8080

# Fetch roles from running nodes (duck or goose)
curl http://localhost:9090/v1/get-role
curl http://localhost:9091/v1/get-role
curl http://localhost:9092/v1/get-role

# list information about cluster
curl http://localhost:9090/v1/nodes

# remove node from cluster
curl -X DELETE http://localhost:9090/v1/nodes/[NODE_ID]
```

## How it works
This program leverages github.com/shaj13/raft to implement a consensus algorithm. We treat the leader as "Goose", and all other nodes as "Duck".

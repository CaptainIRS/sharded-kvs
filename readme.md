# Sharded Key - Value Pair Storage

## Pending Tasks :
- Add mutex while doing file operations
- update/insert for duplicate key = handle it

## Functionalities 
- insert : To insert new key value pair
- select : Get all data for a particular key

- findDestinationServer : client can reach out to any of the server which will automatically send request to destination server in shards

- applied static consistent hashing where 4 servers are situated at 90,180,270,360 in a ring
- used text file as a db
- used grpc for communication

- Multiple client can make requests simultaneously to any of 4 servers.

## Installation
pip install grpcio grpcio-tools

## To compile proto file
python3 -m grpc_tools.protoc --proto_path=. --python_out=. --grpc_python_out=. service.proto 

## Run 4 servers :
- python3 server.py --port 5051
- python3 server.py --port 5052
- python3 server.py --port 5053
- python3 server.py --port 5054

## Run client :
- python3 client.py

## Files :
- server.py : server code
- client.py : client code
- service.proto : all rpc service declaraions
- All server will create their own db text files



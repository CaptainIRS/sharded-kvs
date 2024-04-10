import grpc
from concurrent import futures
import time
import argparse
import sys

import service_pb2
import service_pb2_grpc

my_port = 0000
my_file_db = ""
ring_hashing_mod = 360
server_list = {
    360 : "localhost:5051",
    90 : "localhost:5052",
    180 : "localhost:5053",
    270 : "localhost:5054"
}

# Find in which server key belongs
def findDestinationServer(key):
    server_index = None
    server_address = None
    try:
        new_key = int(key) % ring_hashing_mod
        for dict_key in server_list:
            if dict_key > new_key:
                if server_index is None or dict_key < server_index:
                    server_index = dict_key
        
        if server_index is not None:
            server_address = server_list[server_index]

        return server_index,server_address
    except ValueError:
        return server_index,server_address
    
# Do row search in file to gather all the data matches with input key
def gather_keys_from_file(file_path, search_input):
    response = ""
    with open(file_path, 'r') as file:
        for line in file:
            key, value = line.strip().split('\t')
            if key == search_input:
                response = response + line
    return response
    
    
# All RPC methods for db operations
class DbOperationsServicer(service_pb2_grpc.DbOperationsServicer):
    def Method1(self, request, context):
        print("Received request for Method1 with message:", request.message, "from:", context.peer())
        time.sleep(5)
        return service_pb2.Response1(message="Hello " + request.message)

    def Method2(self, request, context):
        print("Received request for Method2 with message:", request.message, "from:", context.peer())
        return service_pb2.Response2(message="Hi " + request.message)
    
    def InsertData(self, request,context):
        print("Received request for InsertData with message:", request.key , request.value, "from:", context.peer())
        key = request.key
        value = request.value
        try:
            key = int(key)
            server_index,server_address = findDestinationServer(key)
            if server_index is None or server_address is None:
                return service_pb2.ResopnseInsertData(message="Invalid Request")
            # Request belongs to ownself
            if server_address.split(':')[-1] == my_port:
                with open(my_file_db, 'a') as file:
                    line = str(key) + '\t' + value 
                    file.write(line + '\n')
                return service_pb2.ResopnseInsertData(message="Data Inserted Successfully " + "key : " + request.key + " value : " + request.value)
            # Request belongs to some other server make rpc call
            else:
                print("Sending Request to : ",server_address)
                channel = grpc.insecure_channel(server_address)
                stub = service_pb2_grpc.DbOperationsStub(channel)
                request_data = service_pb2.RequestInsertData(key=str(key), value=value)
                response = stub.InsertData(request_data)
                print("Response recieved : ",response.message)
                return service_pb2.ResopnseInsertData(message=response.message)
        except Exception as e:
            return service_pb2.ResopnseInsertData(message="Invalid Request, "+e)
        
    
    def SelectData(self, request,context):
        print("Received request for SelectData with message:", request.key , "from:", context.peer())
        key = request.key
        try:
            key = int(key)
            server_index,server_address = findDestinationServer(key)
            if server_index is None or server_address is None:
                return service_pb2.ResopnseInsertData(message="Invalid Request")
            # Request belongs to ownself
            if server_address.split(':')[-1] == my_port:
                response_data = gather_keys_from_file(my_file_db,str(key))
                return service_pb2.ResopnseInsertData(message=response_data)
            # Request belongs to some other server make rpc call
            else:
                print("Sending Request to : ",server_address)
                channel = grpc.insecure_channel(server_address)
                stub = service_pb2_grpc.DbOperationsStub(channel)
                request_data = service_pb2.RequestSelectData(key=str(key))
                response = stub.SelectData(request_data)
                print("Response recieved : \n",response.message)
                return service_pb2.ResopnseInsertData(message=response.message)
        except Exception as e:
            return service_pb2.ResopnseInsertData(message="Invalid Request, "+e)
        


def serve(port):
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    service_pb2_grpc.add_DbOperationsServicer_to_server(DbOperationsServicer(), server)
    server.add_insecure_port('[::]:' + port)
    server.start()
    print("Server started on port", port)
    # a,b = findDestinationServer(90)
    # print(a,b)
    try:
        while True:
            time.sleep(86400)
    except KeyboardInterrupt:
        server.stop(0)

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Start gRPC server.')
    parser.add_argument('--port', type=str, help='Port number for the gRPC server')
    args = parser.parse_args()

    if args.port is None:
        print("Error: Port number not provided. Please specify the port using --port option.")
        sys.exit(1)

    my_port = args.port 
    my_file_db = str(my_port) + "db.txt"
    serve(args.port)


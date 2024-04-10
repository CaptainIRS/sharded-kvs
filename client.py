import grpc

import service_pb2
import service_pb2_grpc

def run():
    channel = grpc.insecure_channel('localhost:5051')
    stub = service_pb2_grpc.DbOperationsStub(channel)
    
    while True:
        command = input("Type :\nget for testing\ninsert\nselect\n")
        
        if command == 'quit':
            break
        elif command == 'get':
            response1 = stub.Method1(service_pb2.Request1(message='World'))
            print("Method1 response:", response1.message)

            response2 = stub.Method2(service_pb2.Request2(message='Universe'))
            print("Method2 response:", response2.message)
        elif command == 'insert':
            key = input("Insert key in int : ")
            value = input("Input value : ")
            request_data = service_pb2.RequestInsertData(key=key, value=value)
            print(request_data)
            response = stub.InsertData(request_data)
            print("Insert Data : ",response.message)
        elif command == 'select':
            key = input("Insert key in int : ")
            request_data = service_pb2.RequestSelectData(key=key)
            print(request_data)
            response = stub.SelectData(request_data)
            print("Data : \n"+response.message)    
        else:
            print("Invalid command. Please type 'get' to call methods, or 'quit' to exit.")

if __name__ == '__main__':
    run()

import socket

BUFSIZE = 1024

port = ('127.0.0.1', 2233)


server = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
server.bind(port)

while True:
    _, addr = server.recvfrom(BUFSIZE)
    print(addr)
    server.sendto(addr[0]+':'+str(addr[1]).encode('utf-8'), addr)

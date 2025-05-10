# Docker Container Troubleshooting Scenario

## User Query
My Docker containers keep crashing and I'm not sure why. Can you help me troubleshoot?

## Agent Conversation

Cloud-Assist: I'll help you troubleshoot your Docker container crashes. First, let's check which containers are currently running or have recently crashed:

Suggested command:
```bash
docker ps -a
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
CONTAINER ID   IMAGE                  COMMAND                  CREATED          STATUS                     PORTS                                      NAMES
a12b3c4d5e6f   postgres:14           "docker-entrypoint.s…"   15 minutes ago   Up 15 minutes              0.0.0.0:5432->5432/tcp                     db
7g8h9i0j1k2l   redis:alpine          "docker-entrypoint.s…"   15 minutes ago   Up 15 minutes              0.0.0.0:6379->6379/tcp                     cache
3m4n5o6p7q8r   myapp:latest          "node server.js"         15 minutes ago   Exited (1) 5 minutes ago                                              api-service
9s0t1u2v3w4x   nginx:latest          "/docker-entrypoint.…"   15 minutes ago   Up 15 minutes              0.0.0.0:80->80/tcp                         web
5y6z7a8b9c0d   myapp-worker:latest   "npm run worker"         15 minutes ago   Exited (137) 2 minutes ago                                            worker
```

Cloud-Assist: I see a couple of containers that have exited: `api-service` with exit code 1 and `worker` with exit code 137. Let's check the logs for the `api-service` container first:

Suggested command:
```bash
docker logs api-service
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
Starting API service...
Connected to database
Error: ECONNREFUSED connecting to cache at redis:6379
    at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1495:16)
    at Socket.emit (node:events:513:28)
    at Socket.emit (node:domain:489:12)
    at TCPConnectWrap.afterConnectError (node:net:1487:10)
Node server crashed. Exiting with code 1.
```

Cloud-Assist: The `api-service` container is crashing with a connection error to Redis. It's trying to connect to "redis:6379" but can't reach it. Let's check if the containers are on the same network:

Suggested command:
```bash
docker network ls
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
NETWORK ID     NAME                  DRIVER    SCOPE
1a2b3c4d5e     bridge                bridge    local
6f7g8h9i0j     my-app-network        bridge    local
1k2l3m4n5o     host                  host      local
6p7q8r9s0t     none                  null      local
```

Cloud-Assist: Now let's see which network each container is connected to:

Suggested command:
```bash
docker network inspect my-app-network
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```json
[
    {
        "Name": "my-app-network",
        "Id": "6f7g8h9i0j",
        "Created": "2025-05-10T14:30:00.123456789Z",
        "Scope": "local",
        "Driver": "bridge",
        "EnableIPv6": false,
        "IPAM": {
            "Driver": "default",
            "Options": {},
            "Config": [
                {
                    "Subnet": "172.18.0.0/16",
                    "Gateway": "172.18.0.1"
                }
            ]
        },
        "Internal": false,
        "Attachable": false,
        "Ingress": false,
        "ConfigFrom": {
            "Network": ""
        },
        "ConfigOnly": false,
        "Containers": {
            "7g8h9i0j1k2l": {
                "Name": "cache",
                "EndpointID": "1a2b3c4d5e6f7g8h",
                "MacAddress": "02:42:ac:12:00:02",
                "IPv4Address": "172.18.0.2/16",
                "IPv6Address": ""
            },
            "9s0t1u2v3w4x": {
                "Name": "web",
                "EndpointID": "9i0j1k2l3m4n5o6p",
                "MacAddress": "02:42:ac:12:00:04",
                "IPv4Address": "172.18.0.4/16",
                "IPv6Address": ""
            },
            "a12b3c4d5e6f": {
                "Name": "db",
                "EndpointID": "7q8r9s0t1u2v3w4x",
                "MacAddress": "02:42:ac:12:00:03",
                "IPv4Address": "172.18.0.3/16",
                "IPv6Address": ""
            }
        },
        "Options": {},
        "Labels": {}
    }
]
```

Cloud-Assist: I found the issue! The `api-service` and `worker` containers are not connected to the same network as the `cache` (Redis) container. The `api-service` is trying to connect to Redis using the hostname "redis" but it can't reach it because they're not on the same network.

Let's connect the `api-service` container to the correct network:

Suggested command:
```bash
docker network connect my-app-network api-service
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: e

Cloud-Assist: This command connects the `api-service` container to the `my-app-network` bridge network:
- `docker network connect` is used to connect a container to a network
- `my-app-network` is the name of the network to connect to (where your Redis container is running)
- `api-service` is the name of the container we want to connect to the network

By connecting the container to the same network as Redis, the container will be able to resolve the hostname "redis" to the correct IP address, allowing the connection to succeed.

[y] Execute  [n] Skip  [q] Quit

User: y

Output:
```
Error response from daemon: Container 3m4n5o6p7q8r is not running
```

Cloud-Assist: Ah, we need to restart the container first since it's currently in an exited state. Let's start the container again and then connect it to the network:

Suggested command:
```bash
docker start api-service
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
api-service
```

Cloud-Assist: Now let's connect it to the network:

Suggested command:
```bash
docker network connect my-app-network api-service
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
(empty output)
```

Cloud-Assist: Great! Now let's do the same for the worker container:

Suggested command:
```bash
docker start worker
docker network connect my-app-network worker
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
worker
(empty output)
```

Cloud-Assist: Let's check if our containers are running properly now:

Suggested command:
```bash
docker ps -a
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
CONTAINER ID   IMAGE                  COMMAND                  CREATED          STATUS                  PORTS                                      NAMES
a12b3c4d5e6f   postgres:14           "docker-entrypoint.s…"   22 minutes ago   Up 22 minutes           0.0.0.0:5432->5432/tcp                     db
7g8h9i0j1k2l   redis:alpine          "docker-entrypoint.s…"   22 minutes ago   Up 22 minutes           0.0.0.0:6379->6379/tcp                     cache
3m4n5o6p7q8r   myapp:latest          "node server.js"         22 minutes ago   Up 30 seconds                                                       api-service
9s0t1u2v3w4x   nginx:latest          "/docker-entrypoint.…"   22 minutes ago   Up 22 minutes           0.0.0.0:80->80/tcp                         web
5y6z7a8b9c0d   myapp-worker:latest   "npm run worker"         22 minutes ago   Up 15 seconds                                                       worker
```

Cloud-Assist: Excellent! All containers are now running. Let's check the logs of the `api-service` to make sure it's connecting to Redis correctly:

Suggested command:
```bash
docker logs --tail 10 api-service
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
Starting API service...
Connected to database
Connected to Redis cache
API service listening on port 3000
```

Cloud-Assist: Perfect! The API service is now connecting to both the database and Redis cache successfully. The issue was that the containers were running on different Docker networks, so they couldn't communicate with each other by hostname.

Let's also check the worker logs:

Suggested command:
```bash
docker logs --tail 10 worker
```

[y] Execute  [n] Skip  [e] Explain  [q] Quit

User: y

Output:
```
Starting worker...
Connected to message queue
Listening for jobs...
```

Cloud-Assist: Great! Both services are now running properly. 

To summarize the problem and solution:
1. The containers were on different networks, so they couldn't communicate by hostname
2. We connected both containers to the same `my-app-network` where Redis is running
3. This allowed the services to resolve the Redis hostname and connect successfully

To prevent this issue in the future, you should ensure all your containers that need to communicate with each other are on the same Docker network. You could also use Docker Compose to manage your multi-container applications, which automatically creates a shared network for all services.

Would you like me to show you how to set this up with Docker Compose to avoid such issues in the future?

# to build the docker
```bash
sudo docker build -t hw1-part3 .
```

Expected success output:

```text
Successfully tagged hw1-part3:latest
```

---

# Run Docker Container
```bash
sudo docker run -p 8080:8080 hw1-part3
```
Expected output:

```text
[server] Starting server on :8080
```

---

# Testing the Service 

Open a **second terminal** and run tests.
---

1. Health Check

```bash
curl http://localhost:8080/health
```

Expected output:

```text
OK
```

Server log:

```text
[server] Health check request received
```

---

2. ADD Operation

```bash
curl "http://localhost:8080/compute?op=ADD&a=5&b=7"
```

Expected output:

```json
{"status":"ok","operation":"ADD","a":5,"b":7,"result":12}
```

Server log:

```text
[server] Request received: op=ADD a=5.000000 b=7.000000
```

---

3. SUB Operation

```bash
curl "http://localhost:8080/compute?op=SUB&a=10&b=3"
```

Expected:

```json
{"status":"ok","operation":"SUB","a":10,"b":3,"result":7}
```

---

4. MUL Operation

```bash
curl "http://localhost:8080/compute?op=MUL&a=4&b=6"
```

Expected:

```json
{"status":"ok","operation":"MUL","a":4,"b":6,"result":24}
```

---

5. DIV Operation

```bash
curl "http://localhost:8080/compute?op=DIV&a=8&b=2"
```

Expected:

```json
{"status":"ok","operation":"DIV","a":8,"b":2,"result":4}
```

---

# Error Handling Tests

---

## Division By Zero

```bash
curl "http://localhost:8080/compute?op=DIV&a=5&b=0"
```

Expected:

```json
{"status":"err","error":"division_by_zero"}
```

---

## Unknown Operation

```bash
curl "http://localhost:8080/compute?op=POW&a=2&b=3"
```

Expected:

```json
{"status":"err","error":"unknown_operation"}
```

---

## Missing Parameter

```bash
curl "http://localhost:8080/compute?op=ADD&a=5"
```

Expected:

```text
missing parameters
```

---

## Invalid Number

```bash
curl "http://localhost:8080/compute?op=ADD&a=abc&b=5"
```

Expected:

```text
invalid a
```

---

# Verify Running Container

Check running containers:

```bash
sudo docker ps
```

Expected:

```text
CONTAINER ID   IMAGE       COMMAND      CREATED         STATUS         PORTS                                         NAMES       
4bc5bc05b0f8   hw1-part3   "./server"   8 minutes ago   Up 8 minutes   0.0.0.0:8080->8080/tcp, [::]:8080->8080/tcp   silly_nash  
```


---

Stop Container with ctrl + c or command below

```bash
sudo docker stop <container_id>
```

---

# Run Again Later

```bash
cd part3
sudo docker run -p 8080:8080 hw1-part3
```

---

# Rebuild After Code Changes


```bash
docker build -t hw1-part3 .
docker run -p 8080:8080 hw1-part3
```

---

# Testing From Windows Host - in powershell


```powershell
curl http://localhost:8080/health
```

Expected:

```text
OK
```

Compute:

```powershell
curl "http://localhost:8080/compute?op=SUB&a=10&b=3"
```

Expected:

```json
{"status":"ok","operation":"SUB","a":10,"b":3,"result":7}
```


---

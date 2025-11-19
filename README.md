# go-auth

A lightweight microservice that handles authorization logic. It allows you to store access rules (policies) and check if a user has permission to perform an action.

## How to Run

The project is containerized. You do not need to install Go or Postgres locally to run the server.

1.  **Start the Application:**
    ```bash
    docker-compose up --build
    ```

    The server will start on **port 8080**.  
    **Default API Key:** `api-secret`

2.  **Stop the Application:**
    ```bash
    docker-compose down
    ```

## API Usage

### 1. Create a Policy (Protected)
Adds a new rule to the system. Requires the API Key in the header.

**Endpoint:** `POST /policies`  
**Header:** `Authorization: Bearer api-secret`

```bash
curl -X POST http://localhost:8080/policies -H "Content-Type: application/json" -H "Authorization: Bearer api-secret" -d "{\"subject\": \"user:sigma\", \"object\": \"data1\", \"action\": \"read\"}"
```

### 2. Check permission(Public)
Asks the engine if a specific request is allowed.  
**Endpoint:** `POST /check`
* Example:

**If allowed:**
```bash
curl -X POST http://localhost:8080/check -H "Content-Type: application/json" -d "{\"subject\": \"user:sigma\", \"object\": \"data1\", \"action\": \"read\"}"
```
**Response:**
```bash
{
  "allowed": true
}
```

**If denied**
```bash
curl -X POST http://localhost:8080/check -H "Content-Type: application/json" -d "{\"subject\": \"user:sigma\", \"object\": \"data1\", \"action\": \"sleep\"}"
```
**Response:**
```bash
{
  "allowed": false
}
```

  
# TC Go Application

---

**Purpose**  
This application stores, processes, and manages data efficiently.

---

## Quick start quid

```bash
make up
```

## Monitoring

Use [RabbitMQ UI](http://localhost:15672) to monitor queue.

- login: `guest`
- pass: `guest`

## Functionality

1. **Data Storage**:
    - The application creates a `source_data` table with the following fields:
        - `id` (primary key)
        - `name` (string)
        - `description` (text)
        - `created_at` (timestamp)

2. **Data Generation**:
    - Uses the gofakeit package to generate at least 10,000 records in the `source_data` table.

3. **Data Processing**:
    - After inserting data into the `source_data` table, the application sends each record to a RabbitMQ queue for
      processing.

4. **RabbitMQ Integration**:
    - The application connects to RabbitMQ, listens for incoming messages, and processes them by unmarshalling the data.

5. **Destination Storage**:
    - Processed data is saved to the `destination_data` table, which mirrors the `source_data` structure.

6. **Batch Processing**:
    - Messages are processed in batches to enhance performance and reduce database transactions.

7. **Scalability**:
    - The application is designed to scale horizontally by using a worker pool to manage concurrency and efficiently
      handle large volumes of data.

8. **Continuous Listening**:
    - The application remains active, continuously listening for new messages in a loop, ensuring it is always ready to
      process incoming data.

---

## Data Flow

1. The application creates records in the `source_data` table and sends them to RabbitMQ.
2. It listens for messages from RabbitMQ, processes the data, and writes it to the `destination_data` table.
3. By sharing the same database, the application ensures a seamless and efficient data management process.
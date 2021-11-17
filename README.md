# YACA (Yet Another Chat App)

## Example messaging application that can be easily scaled to high traffic.

### Instructions

1. Install Docker.
2. Set `HOST_IP` environment variable with:

   ```bash
   export HOST_IP=$(ifconfig | grep -E "([0-9]{1,3}\.){3}[0-9]{1,3}" | grep -v 127.0.0.1 \
       | awk '{print $2}' | cut -f2 -d: |head -n1)
   ```

2. Bring up the containers with `docker-compose up -d`.
3. Wait about 10 seconds or so while Kafka and Zookeeper start up.
4. Create the Debezium Connector for the Postgres database:

   ```bash
   curl -i -X POST -H "Accept:application/json" -H "Content-Type:application/json" \
        localhost:9090/connectors/ -d @connectors/postgres-connector.json
   ```

5. Run `docker-compose up -d` again, since the `pusher` service depends on the Debezium Connector.

6. Navigate to `localhost:3000`.

7. Type messages into the message box and hit the `Send` button. You should observe messages return to the
   browser. If you open multiple tabs, then you should each message on each tab.

### Technologies

* Postgres database
* Debezium Postgres Connector for events using the Postgres WAL
* Kafka for message passing
* Redis for session management
* Golang for the web server and Kafka consumer

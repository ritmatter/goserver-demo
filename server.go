package main
import (
  "database/sql"
  "flag"
  "fmt"
  _ "github.com/lib/pq"
  "github.com/confluentinc/confluent-kafka-go/kafka"
  "github.com/gorilla/websocket"
  "net/http"
  "log"
  "os"
  "strconv"
)

var db *sql.DB
var enableDatabase bool
var enableKafka bool
var topics []string
var upgrader = websocket.Upgrader {
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
}
var conns []*websocket.Conn
var producer *kafka.Producer
var deliveryChan chan kafka.Event

// Creates a kafka producer.
func CreateProducer(brokerServer string) (*kafka.Producer, error) {
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
  fmt.Println("Got hostname: " + hostname)

  return kafka.NewProducer(&kafka.ConfigMap{
    "bootstrap.servers": brokerServer,
    "client.id": hostname,
    "acks": "all"})
}

// Producers a message with the given producer.
func ProduceMessage(p *kafka.Producer, deliveryChan chan kafka.Event, value string, topic string) error {
  fmt.Println("Producing message: " + value + " for topic: " + topic)
  return p.Produce(&kafka.Message{
      TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
      Value: []byte(value)},
      deliveryChan,
  )
}

// Handle a produce event for a given producer.
// Create a go routine with this func for async writes.
func HandleProduceEvent(p *kafka.Producer) {
  for e := range p.Events() {
      switch ev := e.(type) {
      case *kafka.Message:
          if ev.TopicPartition.Error != nil {
              fmt.Printf("Failed to deliver message: %v\n", ev.TopicPartition)
          } else {
              fmt.Printf("Successfully produced record to topic %s partition [%d] @ offset %v\n",
                  *ev.TopicPartition.Topic, ev.TopicPartition.Partition, ev.TopicPartition.Offset)
          }
      }
  }
}

// Produces a message synchronously. Useful for testing.
func SynchronousProduce(p *kafka.Producer, topic string, value string) {
  delivery_chan := make(chan kafka.Event, 10000)
  err := p.Produce(&kafka.Message{
      TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
      Value: []byte(value)},
      delivery_chan)
  if err != nil {
   panic(err)
  }

  e := <-delivery_chan
  m := e.(*kafka.Message)

  fmt.Println("Examining result of delivery")
  if m.TopicPartition.Error != nil {
    fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
  } else {
    fmt.Printf("Delivered message to topic %s [%d] at offset %v\n",
               *m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
  }
  close(delivery_chan)
}

// Reader that receives web socket messages
func reader(conn *websocket.Conn) {
  for {
    // Read message.
    messageType, p, err := conn.ReadMessage()
    if err != nil {
        log.Println(err)
        return
    }

    // Print message.
    fmt.Println("Received client message: `" + string(p) + "`")

    response := "Ack: " + string(p)
    if err := conn.WriteMessage(messageType, []byte(response)); err != nil {
        log.Println(err)
        return
    }

    if enableKafka {
      // Publish the message so other listeners will get it.
      // TODO: Avoid the sender receiving the message they just sent.
      err = ProduceMessage(producer, deliveryChan, string(p), topics[0])
      if err != nil {
        panic(err)
      }
    }
  }
}

func OpenWebsocket(w http.ResponseWriter, req *http.Request) {
  // Allow incoming request from another domain to connect (prevent CORS error).
  upgrader.CheckOrigin = func(r *http.Request) bool { return true }

  // Upgrade the connection to a web socket.
  ws, err := upgrader.Upgrade(w, req, nil)
  if err != nil {
      log.Println(err)
  }

  // Log statement to show that client connected.
  log.Println("Client Connected")
  conns = append(conns, ws)

  // Write a hello message to the client.
  err = ws.WriteMessage(1, []byte("Greetings client."))
  if err != nil {
      log.Println(err)
  }

  // Permanently read from this websocket.
  reader(ws)
}

func ShowWords(w http.ResponseWriter, req *http.Request) {
  fmt.Println("Inside ShowWords handler")

  if (enableDatabase) {
    sqlStatement := "SELECT * From words"
    rows, err := db.Query(sqlStatement)
    if err != nil {
      panic(err)
    }
    defer rows.Close()


    words := make([]string, 0)
    for rows.Next() {
      var word string
      if err := rows.Scan(&word); err != nil {
        log.Fatal(err)
      }
      words = append(words, word)
    }
    fmt.Fprint(w, words)
  } else {
    fmt.Fprint(w, "Database disabled")
  }
}

func main() {
  fmt.Println("Starting up!")
  enableDatabasePtr := flag.Bool("enableDatabase", false, "Whether to use the database.")
  enableKafkaPtr := flag.Bool("enableKafka", false, "Whether to use Kafka messaging.")
  kafkaBrokerServerPtr := flag.String("kafkaBrokerServer", "host.docker.internal:9092", "The kafka broker information.")
  kafkaTopicPtr := flag.String("kafkaTopic", "AwsChatTopic", "The kafka topic")
  clientDirPtr := flag.String("clientDir", "./client/", "The directory with client files.")
  flag.Parse()

  enableKafka = *enableKafkaPtr
  if (enableKafka) {
    topics = []string{*kafkaTopicPtr}

    producerPtr, err := CreateProducer(*kafkaBrokerServerPtr)
    if err != nil {
      panic(err)
    }
    producer = producerPtr
    go HandleProduceEvent(producer)

    // Uncomment to verify that a single, local publish works.
    // SynchronousProduce(producer, topics[0], "Sync pub")

    // Uncomment to verify that a single async message publish works.
    // deliveryChan := make(chan kafka.Event, 10000)
    // err = ProduceMessage(producer, deliveryChan, "Hello world!", topics[0])
    if err != nil {
      panic(err)
    } else {
      fmt.Println("Successfully produced message.")
    }
    fmt.Println("Initialized Kafka producer.")
  } else {
    fmt.Println("Skipping kafka initialization")
  }

  enableDatabase = *enableDatabasePtr
  if (enableDatabase) {
    host := os.Args[1]
    port, err := strconv.Atoi(os.Args[2])
    if err != nil {
      panic(err)
    }

    user := os.Args[3]
    password := os.Args[4]
    dbname := os.Args[5]

    psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
      "password=%s dbname=%s sslmode=disable",
      host, port, user, password, dbname)
    fmt.Println(psqlInfo)

    database, err := sql.Open("postgres", psqlInfo)
    if err != nil {
      panic(err)
    }
    defer database.Close()
    db = database

    err = db.Ping()
    if err != nil {
      panic(err)
    }
    fmt.Println("Successfully connected to database!")
  } else {
   fmt.Println("Skipping database connection")
  }

  http.Handle("/", http.FileServer(http.Dir(*clientDirPtr)))
  http.HandleFunc("/words", ShowWords)
  http.HandleFunc("/ws", OpenWebsocket)
  err := http.ListenAndServe(":3000", nil)
  fmt.Println("Serving on port 3000")
  if err != nil {
    log.Fatal("ListenAndServe: ", err.Error())
  }
}

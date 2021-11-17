package main
import (
  "bytes"
  "context"
  "encoding/json"
  "flag"
  "fmt"
  "github.com/confluentinc/confluent-kafka-go/kafka"
  "github.com/go-redis/redis/v8"
  "io/ioutil"
  "log"
  "net/http"
  "os"
)

var rdb *redis.Client

var ctx = context.Background()

// Creates a kafka consumer and subscribes to topics.
func CreateConsumer(topics *[]string, brokerServer string, groupId string) (*kafka.Consumer, error) {
  consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
     "bootstrap.servers":    brokerServer,
     "group.id":             groupId,
     "auto.offset.reset":    "smallest"})
  if err != nil {
    return consumer, err
  }

  err = consumer.SubscribeTopics(*topics, nil)
  return consumer, err
}

func main() {
  fmt.Println("Pusher is starting up!")
  kafkaTopicPtr := flag.String("kafkaTopic", "AwsChatTopic", "The kafka topic")
  consumerGroupIdPtr := flag.String("consumerGroupId", "consumer", "The consumer group ID.")
  kafkaBrokerServerPtr := flag.String("kafkaBrokerServer", "host.docker.internal:9092", "The kafka broker information.")
  enableRedisPtr := flag.Bool("enableRedis", false, "Whether to enable Redis session store.")
  sessionStorePtr := flag.String("sessionStoreAddr", "host.docker.internal:6379", "Address of the session store.")
  flag.Parse()

  topics := []string{*kafkaTopicPtr}
  consumer, err := CreateConsumer(&topics, *kafkaBrokerServerPtr, *consumerGroupIdPtr)
  if err != nil {
    panic(err)
  }

  enableRedis := *enableRedisPtr
  if enableRedis {
    rdb = redis.NewClient(&redis.Options{
        Addr:     *sessionStorePtr,
        Password: "", // no password set
        DB:       0,  // use default DB
    })
  }

  run := true
  for run == true {
    ev := consumer.Poll(100)
    switch e := ev.(type) {
      case *kafka.Message:
        fmt.Println("%% Message on %s:\n%s\n", e.TopicPartition, string(e.Value))

        var message_json map[string]interface{}
        json.Unmarshal(e.Value, &message_json)
        payload := message_json["payload"].(map[string]interface{})
        op := payload["op"].(string)
        if op != "c" {
          fmt.Println("Skipping message with op %s", op)
        }

        after := payload["after"].(map[string]interface{})
        text := after["text"].(string)
        fmt.Println("Extracted text: %s", text) 

        if rdb != nil {
          // TODO: Support more chat rooms than just "public".
          addresses,err := rdb.SMembers(ctx, "public").Result()
          if err != nil {
            panic(err)
          }
          fmt.Println(addresses)
          for _, address := range addresses {
           fmt.Println(address)
           postBody, _ := json.Marshal(map[string]string{
              "message": text,
           })
           requestBody := bytes.NewBuffer(postBody)
           resp, err := http.Post("http://" + address, "application/json", requestBody)
           if err != nil {
             fmt.Println(err)
             continue
           }
           defer resp.Body.Close()

           body, err := ioutil.ReadAll(resp.Body)
           if err != nil {
              log.Fatalln(err)
           }
           sb := string(body)
           log.Printf(sb)
          }
        }

      case kafka.PartitionEOF:
          fmt.Printf("%% Reached %v\n", e)
      case kafka.Error:
          fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
          run = false
      default:
    }
  }
  consumer.Close()
}

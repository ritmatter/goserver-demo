package main
import (
  "flag"
  "fmt"
  "github.com/confluentinc/confluent-kafka-go/kafka"
  "os"
)

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
  flag.Parse()

  topics := []string{*kafkaTopicPtr}
  consumer, err := CreateConsumer(&topics, *kafkaBrokerServerPtr, *consumerGroupIdPtr)
  if err != nil {
    panic(err)
  }

  run := true
  for run == true {
    ev := consumer.Poll(1000)
    switch e := ev.(type) {
      case *kafka.Message:
        fmt.Printf("%% Message on %s:\n%s\n", e.TopicPartition, string(e.Value))

        // TODO: Send requests to all the chat servers to push the message.
//        for _, conn := range conns {
//          if err := conn.WriteMessage(websocket.TextMessage, e.Value); err != nil {
//            log.Println(err)
//            return
//          }
//        }

      case kafka.PartitionEOF:
          fmt.Printf("%% Reached %v\n", e)
      case kafka.Error:
          fmt.Fprintf(os.Stderr, "%% Error: %v\n", e)
          run = false
      default:
          fmt.Printf("Ignored %v\n", e)
    }
  }
  consumer.Close()
}

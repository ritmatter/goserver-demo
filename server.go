package main
import (
  "database/sql"
  "flag"
  "fmt"
  _ "github.com/lib/pq"
  "github.com/gorilla/websocket"
  "net/http"
  "log"
  "os"
  "strconv"
)

var db *sql.DB
var enableDatabase bool
var upgrader = websocket.Upgrader {
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
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
    fmt.Println("Received message: `" + string(p) + "`")

    response := "Ack: " + string(p)
    if err := conn.WriteMessage(messageType, []byte(response)); err != nil {
        log.Println(err)
        return
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
  flag.Parse()

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

  http.HandleFunc("/", ShowWords)
  http.HandleFunc("/ws", OpenWebsocket)
  err := http.ListenAndServe(":3000", nil)
  fmt.Println("Serving on port 3000")
  if err != nil {
    log.Fatal("ListenAndServe: ", err.Error())
  }
}

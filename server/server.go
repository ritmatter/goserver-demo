package main
import (
  "context"
  "database/sql"
  "flag"
  "fmt"
  _ "github.com/lib/pq"
  "github.com/google/uuid"
  "github.com/gorilla/websocket"
  "github.com/go-redis/redis/v8"
  "io/ioutil"
  "net"
  "net/http"
  "log"
  "os"
  "strconv"
  "time"
)

var maxConnections int
var db *sql.DB
var rdb *redis.Client
var enableDatabase bool
var topics []string
var upgrader = websocket.Upgrader {
  ReadBufferSize:  1024,
  WriteBufferSize: 1024,
}
var conns map[string]*websocket.Conn

// Request context.
var ctx = context.Background()

// Address of this server.
var address string

// Port accepting message pushes.
var pushPort int

// Define constants for websocket handling.
const (
	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

  // Default channel ID for testing.
  defaultChannelId = "3f333df6-90a4-4fda-8dd3-9485d27cee36"
)

type PushRequest struct {
    message map[string]string
}

// Writes a message into the database.
func PersistMessage(text string) (sql.Result, error)  {
  sqlStatement := `
  INSERT INTO messages (channel_id, text)
  VALUES ($1, $2)`

  return db.Exec(sqlStatement, defaultChannelId, text)
}

// Reader that receives web socket messages
func reader(conn *websocket.Conn, conn_id string) {
  for {
    _, p, err := conn.ReadMessage()
    if err != nil {
        log.Println(err)
        break
    }

    text := string(p)
    if enableDatabase {
      _, err = PersistMessage(text)
      if err != nil {
        panic(err)
      }
    }
  }

  HandleClose(conn, conn_id)
}

// Handles cleanup when a connection is closed.
func HandleClose(conn *websocket.Conn, conn_id string) {
  conn.Close()
  delete (conns, conn_id)

  // TODO: If no longer any listeners for a given channel, remove
  // this server as a participant in the channel.
}

func OpenWebsocket(w http.ResponseWriter, req *http.Request) {
  // Allow incoming request from another domain to connect (prevent CORS error).
  upgrader.CheckOrigin = func(r *http.Request) bool { return true }

  // Upgrade the connection to a web socket.
  ws, err := upgrader.Upgrade(w, req, nil)
  if err != nil {
      log.Println(err)
  }

  // If there are too many connections, end the connection. This is better than
  // rejecting the connection because it allows us to tell the client what happened.
  // See https://stackoverflow.com/questions/21762596/how-to-read-status-code-from-rejected-websocket-opening-handshake-with-javascrip
  if len(conns) >= maxConnections {
   // TODO: Distinguish between app-layer messages and user messages.
    ws.WriteMessage(1, []byte("Maximum connections, please try again later."))
    ws.Close()
    return
  }

	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

  // Log statement to show that client connected.
  log.Println("Client Connected")

  conn_id := uuid.New().String()
  conns[conn_id] = ws

  // If redis is enabled, add the client to the session store.
  // TODO: Support adding for specific chat channels, not just "public".
  if rdb != nil {
    err := rdb.SAdd(ctx, "public", address + ":" + strconv.Itoa(pushPort), 0).Err()
    if err != nil {
        log.Println("Error connecting to redis.")
        panic(err)
    }
  }

  // Write a hello message to the client.
  err = ws.WriteMessage(1, []byte("Greetings client."))
  if err != nil {
    log.Println(err)
  }

  // Permanently read from this websocket.
  reader(ws, conn_id)
}

func HandlePush(w http.ResponseWriter, req *http.Request) {
  defer req.Body.Close()
  body, err := ioutil.ReadAll(req.Body)
  if err != nil {
    panic(err)
  }

  // TODO: Check the channel before pushing to every conn.
  // TODO: Handle concurrency websocket writes, right now
  // there could be two concurrent pushes.
  for _, conn := range conns {
    err = conn.WriteMessage(websocket.TextMessage, []byte(body))
    if err != nil {
      fmt.Println(err)
    }
  }

  fmt.Fprint(w, "Push OK.")
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
  enableRedisPtr := flag.Bool("enableRedis", false, "Whether to enable Redis session store.")
  sessionStorePtr := flag.String("sessionStoreAddr", "host.docker.internal:6379", "Address of the session store.")
  pushPortPtr := flag.Int("pushPort", 8080, "Port on which to accept message pushes")
  userPortPtr := flag.Int("userPort", 3000, "Port on which to accept user traffic")
  maxConnectionsPtr := flag.Int("maxConnections", 3, "Maximum connections to accept")

  // Database args.
  enableDatabasePtr := flag.Bool("enableDatabase", false, "Whether to use the database.")
  databaseHostPtr := flag.String("databaseHost", "", "Database host.")
  databasePortPtr := flag.Int("databasePort", 5432, "Database port.")
  databaseUserPtr := flag.String("databaseUser", "", "Database user.")
  databasePasswordPtr := flag.String("databasePassword", "", "Database password.")
  databaseNamePtr := flag.String("databaseName", "", "Database name.")
  flag.Parse()

  maxConnections = *maxConnectionsPtr

	hostname, err := os.Hostname()
	if err != nil {
    panic(err)
	}
  addr, err := net.LookupIP(hostname)
  if err != nil {
    panic(err)
  }

  if len(addr) < 1 {
    panic("Error no IP addresses found.")
  }

  address = addr[0].String()

  fmt.Println("Got address: " + address)
  pushPort = *pushPortPtr
  fmt.Println("Got push port: " + strconv.Itoa(pushPort))

  conns = make(map[string]*websocket.Conn)

  enableRedis := *enableRedisPtr
  if enableRedis {
    rdb = redis.NewClient(&redis.Options{
        Addr:     *sessionStorePtr,
        Password: "", // no password set
        DB:       0,  // use default DB
    })
  }

  enableDatabase = *enableDatabasePtr
  if (enableDatabase) {
    host := *databaseHostPtr
    port := *databasePortPtr
    user := *databaseUserPtr
    password := *databasePasswordPtr
    dbname := *databaseNamePtr

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

  mxUserHttp := http.NewServeMux()
  mxUserHttp.Handle("/", http.FileServer(http.Dir(".")))
  mxUserHttp.HandleFunc("/words", ShowWords)
  mxUserHttp.HandleFunc("/ws", OpenWebsocket)
  go func() {
    userPort := *userPortPtr
    err := http.ListenAndServe(":" + strconv.Itoa(userPort), mxUserHttp)
    if err != nil {
      log.Fatal("ListenAndServe: ", err.Error())
    }
  }()

  pushPortStr := strconv.Itoa(pushPort)
  mxPushHttp := http.NewServeMux()
  mxPushHttp.HandleFunc("/", HandlePush)
  err = http.ListenAndServe(":" + pushPortStr, mxPushHttp)
  if err != nil {
    log.Fatal("ListenAndServe: ", err.Error())
  }
}
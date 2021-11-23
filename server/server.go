package main
import (
  "context"
  "database/sql"
  "encoding/json"
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

type Connection struct {
  socket *websocket.Conn
  id string
  user_id string
}

func (c Connection) WriteMessage(messageType string, payload string) (err error) {
  socket_message := SocketMessage{messageType, payload}
  out, _ := json.Marshal(socket_message)
  return c.socket.WriteMessage(1, []byte(out))
}

func (c Connection) Close() {
  c.socket.Close()
}

func (c Connection) ReadMessage() (messageType int, p []byte, err error) {
  return c.socket.ReadMessage()
}

var conns map[string]*Connection

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

type Channel struct {
  Id string `json:"id"`
  Name string `json:"name"`
  CreatedAt time.Time `json:"createdAt"`
}

type User struct {
  Id string `json:"id"`
  Name string `json:"name"`
}

type SocketMessage struct {
  Type string `json:"type"`
  Payload string `json:"payload"`
}

// Writes a message into the database.
func PersistMessage(text string, user_id string) (sql.Result, error)  {
  sqlStatement := `
  INSERT INTO messages (channel_id, text, user_id)
  VALUES ($1, $2, $3)`

  return db.Exec(sqlStatement, defaultChannelId, text, user_id)
}

// Reader that receives web socket messages.
func reader(conn *Connection, conn_id string, user_id string) {
  for {
    _, p, err := conn.ReadMessage()
    if err != nil {
        log.Println(err)
        break
    }

    text := string(p)
    if enableDatabase {
      _, err = PersistMessage(text, user_id)
      if err != nil {
        panic(err)
      }
    }
  }

  HandleClose(conn, conn_id)
}

// Handles cleanup when a connection is closed.
func HandleClose(conn *Connection, conn_id string) {
  conn.Close()
  delete (conns, conn_id)

  // TODO: If no longer any listeners for a given channel, remove
  // this server as a participant in the channel.
}

func OpenWebsocket(w http.ResponseWriter, req *http.Request) {
  // Allow incoming request from another domain to connect (prevent CORS error).
  upgrader.CheckOrigin = func(r *http.Request) bool { return true }

  user_id := req.URL.Query().Get("userId")
  if user_id == "" {
      http.Error(w, "userId query param must be provided.", http.StatusBadRequest)
      return
  }

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

  connection := new(Connection)
  connection.id = conn_id
  connection.socket = ws
  conns[conn_id] = connection

  // If redis is enabled, add the client to the session store.
  // TODO: Support adding for specific chat channels, not just "public".
  if rdb != nil {
    err := rdb.SAdd(ctx, "public", address + ":" + strconv.Itoa(pushPort), 0).Err()
    if err != nil {
        log.Println("Error connecting to redis.")
        panic(err)
    }
  }

  // Permanently read from this websocket.
  reader(connection, conn_id, user_id)
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
    err = conn.WriteMessage("new_message", string(body))
    if err != nil {
      fmt.Println(err)
    }
  }
}

func ListChannels(w http.ResponseWriter, req *http.Request) {
  if !enableDatabase {
    return
  }

  sqlStatement := "SELECT * From channels;"
  rows, err := db.Query(sqlStatement)
  if err != nil {
    panic(err)
  }
  defer rows.Close()

  channels := make([]Channel, 0)
  for rows.Next() {
    var channel Channel
    if err := rows.Scan(&channel.Id, &channel.Name, &channel.CreatedAt); err != nil {
      log.Fatal(err)
    }
    channels = append(channels, channel)
  }

  out, _ := json.Marshal(channels)
  fmt.Fprint(w, string(out))
}

func CreateUser(w http.ResponseWriter, req *http.Request) {
  if !enableDatabase {
    return
  }

  var user User

  // Try to decode the request body into the struct. If there is an error,
  // respond to the client with the error message and a 400 status code.
  dec := json.NewDecoder(req.Body)
  dec.DisallowUnknownFields()
  err := dec.Decode(&user)
  if err != nil {
      http.Error(w, err.Error(), http.StatusBadRequest)
      return
  }

  sqlStatement := `
  INSERT INTO users (name)
  VALUES ($1)`

  _, err = db.Exec(sqlStatement, user.Name)
  if err != nil {
    fmt.Fprint(w, err)
  }
}


func CreateChannel(w http.ResponseWriter, req *http.Request) {
  if !enableDatabase {
    return
  }

  var channel Channel

  // Try to decode the request body into the struct. If there is an error,
  // respond to the client with the error message and a 400 status code.
  dec := json.NewDecoder(req.Body)
  dec.DisallowUnknownFields()
  err := dec.Decode(&channel)
  if err != nil {
      http.Error(w, err.Error(), http.StatusBadRequest)
      return
  }

  sqlStatement := `
  INSERT INTO channels (name)
  VALUES ($1)`

  _, err = db.Exec(sqlStatement, channel.Name)
  if err != nil {
    fmt.Fprint(w, err)
  }
}

func main() {
  enableRedisPtr := flag.Bool("enableRedis", false, "Whether to enable Redis session store.")
  sessionStorePtr := flag.String("sessionStoreAddr", "localhost:6379", "Address of the session store.")
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

  conns = make(map[string]*Connection)

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
  mxUserHttp.Handle("/", http.FileServer(http.Dir("./client/build")))
  mxUserHttp.HandleFunc("/ws", OpenWebsocket)
  mxUserHttp.HandleFunc("/channels", ListChannels)
  mxUserHttp.HandleFunc("/channels/new", CreateChannel)
  mxUserHttp.HandleFunc("/users/new", CreateUser)
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

package main
import (
  "database/sql"
  "fmt"
  _ "github.com/lib/pq"
  "net/http"
  "log"
  "os"
  "strconv"
)

var db *sql.DB

func ShowWords(w http.ResponseWriter, req *http.Request) {
  fmt.Println("Inside ShowWords handler")

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
}

func main() {
  fmt.Println("Starting up!")
  fmt.Println(os.Args)

  host := os.Args[1]
  port, porterr := strconv.Atoi(os.Args[2])
  if porterr != nil {
    panic(porterr)
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

  fmt.Println("Successfully connected!")
  http.HandleFunc("/",ShowWords)
  err = http.ListenAndServe(":3000", nil)
  fmt.Println("Serving on port 3000")
  if err != nil {
    log.Fatal("ListenAndServe: ", err.Error())
  }
}

package main

import (
  "net/http"
  "github.com/bankole7782/flaarum"
  "github.com/bankole7782/pankul"
  "os"
  "github.com/gorilla/mux"
  "strconv"
)


func main() {
	addr := os.Getenv("PK_FLAARUM_ADDR")
  keyStr := os.Getenv("PK_FLAARUM_KEYSTR")
  projName := os.Getenv("PK_FLAARUM_PROJ")

  cl := flaarum.NewClient(addr, keyStr, projName)
	if err := cl.Ping(); err != nil {
    panic(err)
  }

  // FORMS814 setup. Very important
  pankul.FRCL = cl

  pankul.Admins = []int64{1, }

  // This sample makes use of environment variables to get the current user. Real life application
  // could save a random string to the browser cookies. And this random string point to a userid
  // in the database.
  // The function accepts http.Request as argument which can be used to get the cookies.
  pankul.GetCurrentUser = func(r *http.Request) (int64, error) {
    userid := os.Getenv("USERID")
    if userid == "" {
      return 0, nil
    }
    useridInt64, err := strconv.ParseInt(userid, 10, 64)
    if err != nil {
      return 0, err
    }
    return useridInt64, nil
  }

  // pankul.BaseTemplate = "basetemplate.html"
  r := mux.NewRouter()
  pankul.AddHandlers(r)

  http.ListenAndServe(":3001", r)
}

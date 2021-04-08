package pankul

import (
	"net/http"
	"github.com/gorilla/mux"
)

var FILENAME_SEPARATOR = "____"

func serveJS(w http.ResponseWriter, r *http.Request) {
  vars := mux.Vars(r)
  lib := vars["library"]

  if lib == "jquery" {
    http.ServeFile(w, r, "pankul_files/jquery-3.3.1.min.js")
  } else if lib == "autosize" {
    http.ServeFile(w, r, "pankul_files/autosize.min.js")
  }
}

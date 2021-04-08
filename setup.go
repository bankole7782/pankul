package pankul

import (
  "fmt"
  "net/http"
  "github.com/gorilla/mux"
  "net/url"
  "strings"
  "html/template"
  "github.com/bankole7782/flaarum"
  "github.com/bankole7782/flaarum/flaarum_shared"
)

var FRCL flaarum.Client
var Admins []int64
var GetCurrentUser func(r *http.Request) (int64, error)
var BaseTemplate string

type ExtraCode struct {
  ValidationFn func(postForm url.Values) string
  AfterCreateFn func(id int64)
  AfterUpdateFn func(id int64)
  BeforeDeleteFn func(id int64)
  CanCreateFn func() string
}

var ExtraCodeMap = make(map[int64]ExtraCode)

var BucketName string


func findIn(container []string, toFind string) int {
  for i, inContainer := range container {
    if inContainer == toFind {
      return i
    }
  }
  return -1
}


// lifted from flaarum
func formatTableStruct(tableStruct flaarum_shared.TableStruct) string {
  stmt := "table: " + tableStruct.TableName + "\n"
  stmt += "table_type: " + tableStruct.TableType + "\n"
  stmt += "fields:\n"
  for _, fieldStruct := range tableStruct.Fields {
    stmt += "\t" + fieldStruct.FieldName + " " + fieldStruct.FieldType
    if fieldStruct.Required {
      stmt += " required"
    }
    if fieldStruct.Unique {
      stmt += " unique"
    }
    stmt += "\n"
  }
  stmt += "::\n"
  if len(tableStruct.ForeignKeys) > 0 {
    stmt += "foreign_keys:\n"
    for _, fks := range tableStruct.ForeignKeys {
      stmt += "\t" + fks.FieldName + " " + fks.PointedTable + " " + fks.OnDelete + "\n"
    }
    stmt += "::\n"
  }

  if len(tableStruct.UniqueGroups) > 0 {
    stmt += "unique_groups:\n"
    for _, ug := range tableStruct.UniqueGroups {
      stmt += "\t" + strings.Join(ug, " ") + "\n"
    }
    stmt += "::\n"
  }

  return stmt
}


func createOrUpdateTable(stmt string) error {
	tables, err := FRCL.ListTables()
	if err != nil {
		return err
	}

	tableStruct, err := flaarum_shared.ParseTableStructureStmt(stmt)
	if err != nil {
		return err
	}
	if findIn(tables, tableStruct.TableName) == -1 {
		// table doesn't exist
		err = FRCL.CreateTable(stmt)
		if err != nil {
			return err
		}
	} else {
		// table exists check if it needs update
    currentVersionNum, err := FRCL.GetCurrentTableVersionNum(tableStruct.TableName)
    if err != nil {
      return err
    }

		oldStmt, err := FRCL.GetTableStructure(tableStruct.TableName, currentVersionNum)
		if err != nil {
			return err
		}

		if oldStmt != formatTableStruct(tableStruct) {
			err = FRCL.UpdateTableStructure(stmt)
			if err != nil {
				return err
			}
		}

	}
	return nil
}


func pankulSetup(w http.ResponseWriter, r *http.Request) {
  if err := FRCL.Ping(); err != nil {
    errorPage(w, err.Error())
    return
  }

  if Admins == nil {
    errorPage(w, "You have not set the \"pankul.Admins\". Please set this to a list of ids (in int64) of the Admins of this site.")
    return
  }

  if GetCurrentUser == nil {
    errorPage(w, "You must set the \"pankul.GetCurrentUser\". Please set this variable to a function with signature func(r *http.Request) (int64, error).")
    return
  }

  if BucketName == "" {
    errorPage(w, "You must set the \"pankul.BucketName\". Create a bucket on google cloud and set it to this variable.")
    return
  }

  // create forms general table
  err := createOrUpdateTable(`
  	table: pk_document_structures
  	fields:
  		fullname string required unique
  		tbl_name string required unique
  		comment text
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  // create fields table
  err = createOrUpdateTable(`
  	table: pk_fields
  	fields:
  		dsid int required
  		label string required
  		name string required
  		type string required
  		options string
  		other_options string
  		view_order int
  	::
  	foreign_keys:
  		dsid pk_document_structures on_delete_delete
  	::
  	unique_groups:
  		dsid name
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  err = createOrUpdateTable(`
  	table: pk_files_for_delete
  	fields:
  		created_by int required
  		filepath string required
  	::
  	`)
  if err != nil {
    errorPage(w, err.Error())
    return
  }

  fmt.Fprintf(w, "Setup Completed.")
}


func AddHandlers(r *mux.Router) {

  // Please don't change the paths.

  // Please call this link first to do your setup.
  r.HandleFunc("/pk/setup", pankulSetup)

  // admin pages
  r.HandleFunc("/pk/page", pankulPage)

  // document structure links
  r.HandleFunc("/pk/new-document-structure/", newDocumentStructure)
  r.HandleFunc("/pk/list-document-structures/", listDocumentStructures)
  r.HandleFunc("/pk/delete-document-structure/{document-structure}/", deleteDocumentStructure)
  r.HandleFunc("/pk/view-document-structure/{document-structure}/", viewDocumentStructure)
  r.HandleFunc("/pk/light-edit-document-structure/{document-structure}/", lightEditDocumentStructure)
  r.HandleFunc("/pk/update-document-structure-name/{document-structure}/", updateDocumentStructureName)
  r.HandleFunc("/pk/update-comment/{document-structure}/", updateComment)
  r.HandleFunc("/pk/update-field-labels/{document-structure}/", updateFieldLabels)
  r.HandleFunc("/pk/new-ds-from-template/{document-structure}/", newDSFromTemplate)
  r.HandleFunc("/pk/full-edit-document-structure/{document-structure}/", fullEditDocumentStructure)

  // file links
  r.HandleFunc("/pk/serve-js/{library}/", serveJS)
}


func pankulPage(w http.ResponseWriter, r *http.Request) {
  truthValue, err := isUserAdmin(r)
  if err != nil {
    errorPage(w, err.Error())
    return
  }
  if ! truthValue {
    errorPage(w, "You are not an admin here. You don't have permissions to view this page.")
    return
  }

  tmpl := template.Must(template.ParseFiles(getBaseTemplate(), "pankul_files/pankul-page.html"))
  tmpl.Execute(w, nil)
}

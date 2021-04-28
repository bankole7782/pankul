package pankul

import (
  "github.com/pkg/errors"
  "fmt"
  "github.com/bankole7782/flaarum"
  "time"
  "math"
  "html"
  "strings"
)


type Row struct {
  Id string
  DocAndStructures []DocAndStructure
  NeedsUpdate bool
}


// This function is used for list pages.
// Sample input:
//
//    map[string]string {
//      email = "jj@jj.com",
//      order_by = "dt",
//      order_by = "asc",
//    }
// pageNum starts from one
// It returns the documents in []Row, the total pages as its second output and lastly errors.
func GetDocuments(userId int64, documentStructure string, elementValues map[string]string, pageNum int64) ([]Row, int64, error) {
  userIdInt64 := userId

  detv, err := docExists(documentStructure)
  if err != nil {
    return nil, 0, errors.Wrap(err, "pankul error")
  }
  if detv == false {
    return nil, 0, errors.New(fmt.Sprintf("The document structure '%s' does not exists.", documentStructure))
  }

  if pageNum == 0 {
    return nil, 0, errors.New(fmt.Sprintf("The page numbering starts from '1'."))
  }

  tblName, err := tableName(documentStructure)
  if err != nil {
    return nil, 0, errors.Wrap(err, "pankul error")
  }

  dds, err := GetDocData(documentStructure)
  if err != nil {
    return nil, 0, errors.Wrap(err, "flaarum error")
  }


  whereFragmentParts := make([]string, 0)
  for _, dd := range dds {

    if _, ok := elementValues[dd.Name]; !ok {
      continue
    }

    switch dd.Type {
    case "Text", "Data", "Email", "Read Only", "URL", "Select", "Date", "Datetime":
      data := fmt.Sprintf("'%s'", html.EscapeString(elementValues[dd.Name]))
      whereFragmentParts = append(whereFragmentParts, dd.Name + " = " + data)
    case "Check":
      var data string
      if elementValues[dd.Name] == "on" {
        data = "t"
      } else {
        data = "f"
      }
      whereFragmentParts = append(whereFragmentParts, dd.Name + " = " + data)
    default:
      data := html.EscapeString(elementValues[dd.Name])
      whereFragmentParts = append(whereFragmentParts, dd.Name + " = " + data)
    }
  }

  if _, ok := elementValues["created_by"]; ok {
    whereFragmentParts = append(whereFragmentParts, "created_by = " + html.EscapeString(elementValues["created_by"]))
  }
  if _, ok := elementValues["creation-year"]; ok {
    whereFragmentParts = append(whereFragmentParts, "created_year = " + html.EscapeString(elementValues["creation-year"]))
  }
  if _, ok := elementValues["creation-month"]; ok {
    whereFragmentParts = append(whereFragmentParts, "created_month = " + html.EscapeString(elementValues["creation-month"]))
  }
  whereFragment := strings.Join(whereFragmentParts, "\nand ")

  countStmt := fmt.Sprintf(`
    table: %s
    where:
      %s
    `, tblName, strings.Join(whereFragmentParts, "\nand "))

  count, err := FRCL.CountRows(countStmt)
  if err != nil {
    return nil, 0, errors.Wrap(err, "flaarum error")
  }
  if count == 0 {
    return make([]Row, 0), 0, nil
  }

  var itemsPerPage int64 = 50
  startIndex := (pageNum - 1) * itemsPerPage
  totalItems := count
  totalPages := math.Ceil( float64(totalItems) / float64(itemsPerPage) )

  var orderByFragment string
  rowName, ok1 := elementValues["order_by"]
  direction, ok2 := elementValues["direction"]
  if ok1 && ok2 {
    orderByFragment = fmt.Sprintf(" %s %s", rowName, direction)
  } else {
    orderByFragment = " id desc"
  }

  rows, err := FRCL.Search(fmt.Sprintf(`
    table: %s
    order_by: %s
    limit: %d
    start_index: %d
    where:
      %s
    `, tblName, orderByFragment, itemsPerPage, startIndex, whereFragment))
  if err != nil {
    return nil, 0, errors.Wrap(err, "flaarum error")
  }

  currentVersionNum, err := FRCL.GetCurrentTableVersionNum(tblName)
  if err != nil {
    return nil, 0, errors.Wrap(err, "flaarum error")
  }

  myRows := make([]Row, 0)
  for _, rowMapItem := range *rows {
    var needsUpdate bool
    if currentVersionNum == rowMapItem["_version"].(int64) {
      needsUpdate = false
    } else {
      needsUpdate = true
    }

    row := Row {Id: rowMapItem["id"].(string), NeedsUpdate: needsUpdate}
    docAndStructureSlice := make([]DocAndStructure, 0)
    for _, docData := range dds {
      var data string
      switch dInType := rowMapItem[docData.Name].(type) {
      case int64, float64:
        data = fmt.Sprintf("%v", dInType)
      case time.Time:
        if docData.Type == "Date" {
          data = dInType.Format("2006-01-02")
        } else {
          dInTypeCorrected, err := timeInUserTimeZone(dInType, userIdInt64)
          if err != nil {
            return nil, 0, errors.Wrap(err, "time error")
          }
          data = flaarum.RightDateTimeFormat(dInTypeCorrected)
        }
      case string:
        data = dInType
      case bool:
        data = boolToStr(dInType)
      }

      docAndStructureSlice = append(docAndStructureSlice, DocAndStructure{docData, data})
    }

    row.DocAndStructures = docAndStructureSlice
    myRows = append(myRows, row)
  }

  return myRows, int64(totalPages), nil
}

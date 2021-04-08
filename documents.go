package pankul

import (
  "github.com/pkg/errors"
  "net/http"
  "fmt"
  "github.com/bankole7782/flaarum"
  "time"
  "strconv"
)


func SaveDocument(ds string, r *http.Request) (int64, error) {
  userIdInt64, err := GetCurrentUser(r)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }

  detv, err := docExists(ds)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }
  if detv == false {
    return 0, errors.New(fmt.Sprintf("The document structure %s does not exists.", ds))
  }

  dds, err := GetDocData(ds)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }

  tblName, err := tableName(ds)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }
  toInsert := make(map[string]string)
  for _, dd := range dds {
    if dd.Type == "Section Break" {
      continue
    }
    switch dd.Type {
    case "Check":
      var data string
      if r.FormValue(dd.Name) == "on" {
        data = "t"
      } else {
        data = "f"
      }
      toInsert[dd.Name] = data

    case "Datetime":
      if r.FormValue(dd.Name) != "" {
        tzname, err := getUserTimeZoneSuffix(userIdInt64)
        if err != nil {
          return 0, errors.Wrap(err, "pankul error")
        }
        toInsert[dd.Name] = r.FormValue(dd.Name) + " " + tzname
      }

    default:
      if r.FormValue(dd.Name) != "" {
        toInsert[dd.Name] = r.FormValue(dd.Name)
      }
    }
  }

  toInsert["created"] = flaarum.RightDateTimeFormat(time.Now())
  toInsert["modified"] = flaarum.RightDateTimeFormat(time.Now())
  toInsert["created_by"] = fmt.Sprintf("%d", userIdInt64)

  lastIdStr, err := FRCL.InsertRowStr(tblName, toInsert)
  if err != nil {
    return 0, errors.Wrap(err, "pankul error")
  }

  lastId, err := strconv.ParseInt(lastIdStr, 10, 64)
  if err != nil {
    return 0, errors.Wrap(err, "strconv error")
  }

  return lastId, nil
}

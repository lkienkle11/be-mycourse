package domain

import "encoding/json"

// jsonUnmarshal is a thin wrapper kept here so the domain imports encoding/json once.
var jsonUnmarshal = json.Unmarshal

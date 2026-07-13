package orchestrator_graph

import (
	"fmt"
	"regexp"
	"strings"
)

type ResourceCoordinate struct {
	Type  string
	Class string
	Id    string
}

func (rc ResourceCoordinate) String() string {
	return fmt.Sprintf("type=%s,class=%s,id=%s", rc.Type, rc.Class, rc.Id)
}

func CompareResourceCoordinate(a, b ResourceCoordinate) int {
	if i := strings.Compare(a.Type, b.Type); i != 0 {
		return i
	} else if i = strings.Compare(a.Class, b.Class); i != 0 {
		return i
	} else {
		return strings.Compare(a.Id, b.Id)
	}
}

func (rc ResourceCoordinate) MarshalText() ([]byte, error) {
	return []byte(rc.String()), nil
}

var rcRegex = regexp.MustCompile(`^type=([^,]+),class=([^,]+),id=(.+)$`)

func (rc *ResourceCoordinate) UnmarshalText(data []byte) error {
	if m := rcRegex.FindStringSubmatch(string(data)); m != nil {
		rc.Type = m[1]
		rc.Class = m[2]
		rc.Id = m[3]
		return nil
	}
	return fmt.Errorf("invalid resource coordinate: %s", string(data))
}

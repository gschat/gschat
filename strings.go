package gschat

import "fmt"

func (named *NamedService) String() string {
	return fmt.Sprintf("%s-%s:%d", named.Name, named.Type, named.VNodes)
}

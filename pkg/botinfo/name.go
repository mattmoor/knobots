package botinfo

import (
	"os"
)

func GetName() string {
	return os.Getenv("K_SERVICE")
}

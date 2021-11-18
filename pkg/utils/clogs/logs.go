package clogs

import "fmt"

func GenerateKey(name string, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

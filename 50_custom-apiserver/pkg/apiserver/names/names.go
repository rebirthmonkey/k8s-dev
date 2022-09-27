package names

import (
	"errors"
	"fmt"

	"github.com/dlclark/regexp2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
)

const (
	suffixLength = 8
)

var (
	DefaultNameGenerator = defaultNameGenerator{}
)

type defaultNameGenerator struct{}

func (defaultNameGenerator) GenerateName(base string) string {
	return fmt.Sprintf("%s%s", base, GenerateName())
}

func GenerateName() string {
	return utilrand.String(suffixLength)
}

func DefaultNamingCheck(base string, object metav1.Object) error {
	gn := object.GetGenerateName()
	n := object.GetName()
	if n == "" && gn != "" && gn != base+"-" {
		return errors.New(fmt.Sprintf("resource generate name must be %s-", base))
	}
	regex := fmt.Sprintf("^%s\\-[a-zA-Z0-9]{%d}$", base, suffixLength)
	reg, _ := regexp2.Compile(regex, regexp2.None)
	matches, _ := reg.MatchString(n)
	if !matches {
		return errors.New(fmt.Sprintf("resource name must start with %s- and suffix with %d characters", base, suffixLength))
	}

	return nil
}

package ktypes

import (
	"errors"
	"fmt"
	"strings"
)

type NamespaceKind struct {
	Namespace string
	Kind      Kind
}

func NewNamespaceKind(namespace string, kind Kind) NamespaceKind {
	return NamespaceKind{
		Namespace: namespace,
		Kind:      kind,
	}
}

func (n NamespaceKind) String() string {
	return fmt.Sprintf("%s-%s", n.Namespace, n.Kind)
}

func ParseNamespaceKind(s string) (*NamespaceKind, error) {
	strs := strings.Split(s, "-")
	if len(strs) != 2 {
		return nil, errors.New("invalid string")
	}

	return &NamespaceKind{
		Namespace: strs[0],
		Kind:      Kind(strs[1]),
	}, nil
}

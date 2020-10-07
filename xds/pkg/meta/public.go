package meta

import "github.com/kage-cloud/kage/annos"

const (
	DomainBase   = "kage.cloud"
	DomainCanary = "canary." + DomainBase
	DomainXds    = "xds." + DomainBase
)

func ToMap(a Annotation) map[string]string {
	return annos.ToMap(a.GetDomain(), a)
}

func FromMap(m map[string]string, a Annotation) error {
	return annos.FromMap(a.GetDomain(), m, a)
}

func Merge(m map[string]string, a Annotation) map[string]string {
	annoMap := ToMap(a)
	return MergeMaps(m, annoMap)
}

func Contains(m map[string]string, submap map[string]string) bool {
	for k := range submap {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

func MergeMaps(m1, m2 map[string]string) map[string]string {
	out := map[string]string{}

	for k, v := range m1 {
		out[k] = v
	}

	for k, v := range m2 {
		out[k] = v
	}

	return out
}

type Annotation interface {
	GetDomain() string
}

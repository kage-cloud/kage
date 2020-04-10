package consts

const (
	Domain                     = "cloud.kage"
	LabelValueResourceSnapshot = "snapshot"
	LabelValueResourceKageMesh = "mesh"
	LabelValueResourceCanary   = "canary"
)

const (
	LabelKeyDomain     = "domain"
	LabelKeyResource   = Domain + "/resource"
	LabelKeyLockedDown = Domain + "/locked-down"
	LabelKeyTarget     = Domain + "/target"
	LabelKeyCanary     = Domain + "/canary"
)

const (
	AnnotationKeyLockdown = Domain + "/lockdown"
	AnnotationKageMesh    = Domain + "/kage-mesh"
)

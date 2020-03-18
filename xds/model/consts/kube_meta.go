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
	LabelKeyFor        = Domain + "/for"
)

const (
	AnnotationKeyLockdown = Domain + "/lockdown"
)

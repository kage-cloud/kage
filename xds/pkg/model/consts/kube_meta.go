package consts

const (
	Domain                     = "kage.cloud"
	CanaryDomain               = "canary." + Domain
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
	AnnotationMeshConfig  = Domain + "/mesh-config"
)

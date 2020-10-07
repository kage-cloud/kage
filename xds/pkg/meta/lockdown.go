package meta

type Lockdown struct {
	LockedDown bool
}

func (l Lockdown) GetDomain() string {
	return DomainCanary
}

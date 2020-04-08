package model

type Lockdown struct {
	DeletedSet map[string]string `json:"deleted_labels"`
}

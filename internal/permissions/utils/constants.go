package utils

const (
	Create = "create"
	Update = "update"
	Delete = "delete"
	View   = "view"
)

type PermArr []string

var AdminPermArr PermArr = PermArr{
	Create,
	Update,
	Delete,
	View,
}

package dto

type FindManyRequest struct {
	Limit int `query:"limit"`
	Page  int `query:"page"`
}

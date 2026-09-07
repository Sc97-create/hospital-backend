package dto

type FindManyRequest struct {
	Limit          int    `query:"limit" json:"limit"`
	PageNo         int    `query:"page_no" json:"page_no"`
	OrganisationID string `query:"organisation_id" json:"organisation_id"`
	Search         string `query:"search" json:"search"`
}

package admins

import "errors"

type OrganisationScheduleRepository interface {
	Create(schedule *OrganisationSchedule) error
	GetByOrganisationID(query string, cond ...any) (OrganisationSchedule, error)
}

func (r *CommonDB) Create(schedule *OrganisationSchedule) error {
	return r.db.Create(schedule).Error
}
func (r *CommonDB) GetByOrganisationID(query string, cond ...any) (OrganisationSchedule, error) {
	var orgSched OrganisationSchedule
	err := r.db.Raw(query, cond...).First(&orgSched).Error
	if err != nil {
		return OrganisationSchedule{}, errors.New("failed to fetch data")
	}
	return orgSched, nil
}

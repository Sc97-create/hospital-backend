package organisations

import (
	"strings"

	"hospital-backend/pkg/constants"
)

var allowedOrganisationTypes = map[string]struct{}{
	"hospital": {},
	"clinic":   {},
}

var allowedOrganisationStatuses = map[string]struct{}{
	constants.StatusActive:   {},
	constants.StatusInactive: {},
	"suspended":              {},
}

func isAllowedOrganisationType(v string) bool {
	_, ok := allowedOrganisationTypes[strings.ToLower(strings.TrimSpace(v))]
	return ok
}

func isAllowedOrganisationStatus(v string) bool {
	_, ok := allowedOrganisationStatuses[strings.ToLower(strings.TrimSpace(v))]
	return ok
}

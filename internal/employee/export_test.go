package employee

import "hospital-backend/internal/employee/dto"

// Test hooks for employee_test (avoids package employee internal tests + mocks import cycle).
func (s *EmployeeService) TestGetPageSkip(limit, pageNo int) (int, int) {
	return s.getPageSkip(limit, pageNo)
}

func (s *EmployeeService) TestMapToEmployeeResponse(row EmployeeListRow) dto.EmployeeResponse {
	return s.mapToEmployeeResponse(row)
}

func (s *EmployeeService) TestCreateEmployeeCode(organisationID, dateOfJoining string) (string, error) {
	return s.createEmployeeCode(organisationID, dateOfJoining)
}

func NewEmployeeServiceForTest(repo EmployeeRepository) *EmployeeService {
	return &EmployeeService{EmpRepo: repo}
}

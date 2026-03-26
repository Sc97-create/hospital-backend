package department

type DepartmentService struct {
	DepRepo DepartmentRepository
}

func NewDepartmentService(repo DepartmentRepository) *DepartmentService {
	return &DepartmentService{DepRepo: repo}
}

func (DeptService *DepartmentService) FindMany(limit int, offset int) ([]Department, error) {
	return DeptService.DepRepo.FindMany(limit, offset)
}

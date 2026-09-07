package dashboard

type DashboardContainer struct {
	Service *DashboardService
}

func NewDashboardContainer(
	appointments AppointmentDashboardReader,
	billing BillingDashboardReader,
	employees EmployeeDashboardReader,
	prescriptions PrescriptionDashboardReader,
) *DashboardContainer {
	return &DashboardContainer{
		Service: NewDashboardService(appointments, billing, employees, prescriptions),
	}
}

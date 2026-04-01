package models

// LeadRequestStatus represents the lifecycle status of a public demo request.
type LeadRequestStatus string

const (
	LeadRequestStatusNew       LeadRequestStatus = "new"
	LeadRequestStatusContacted LeadRequestStatus = "contacted"
	LeadRequestStatusQualified LeadRequestStatus = "qualified"
	LeadRequestStatusClosed    LeadRequestStatus = "closed"
)

var validLeadRequestPlans = map[string]struct{}{
	"starter":    {},
	"growth":     {},
	"dedicated":  {},
	"enterprise": {},
}

var validLeadRequestStatuses = map[LeadRequestStatus]struct{}{
	LeadRequestStatusNew:       {},
	LeadRequestStatusContacted: {},
	LeadRequestStatusQualified: {},
	LeadRequestStatusClosed:    {},
}

// LeadRequest stores a public-facing demo/trial request submitted from the pricing page.
type LeadRequest struct {
	BaseModel
	FullName      string            `gorm:"size:255;not null;index" json:"full_name"`
	CompanyName   string            `gorm:"size:255;not null;index" json:"company_name"`
	WorkEmail     string            `gorm:"size:255;not null;index" json:"work_email"`
	PhoneWhatsApp string            `gorm:"size:100;not null" json:"phone_whatsapp"`
	Country       string            `gorm:"size:120" json:"country,omitempty"`
	Message       string            `gorm:"type:text" json:"message,omitempty"`
	RequestedPlan string            `gorm:"size:32;index" json:"requested_plan,omitempty"`
	SourcePage    string            `gorm:"size:50;not null" json:"source_page"`
	SourceRoute   string            `gorm:"size:100;not null" json:"source_route"`
	Status        LeadRequestStatus `gorm:"size:20;not null;default:'new';index" json:"status"`
}

func (LeadRequest) TableName() string {
	return "lead_requests"
}

func IsValidLeadRequestPlan(plan string) bool {
	_, ok := validLeadRequestPlans[plan]
	return ok
}

func IsValidLeadRequestStatus(status LeadRequestStatus) bool {
	_, ok := validLeadRequestStatuses[status]
	return ok
}

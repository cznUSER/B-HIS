package domain

import "time"

// Sex and card status values match OpenAPI 0.3.5 and migrations/sqlserver/core/002_patient.
const (
	SexMale    = "M"
	SexFemale  = "F"
	SexUnknown = "U"

	CardStatusNotSent   = "NOT_SENT"
	CardStatusNormal    = "NORMAL"
	CardStatusLost      = "LOST"
	CardStatusCancelled = "CANCELLED"

	PatientStatusActive   = "ACTIVE"
	PatientStatusInactive = "INACTIVE"
)

// Patient is the outpatient MPI record. Birthday is a calendar date.
type Patient struct {
	PatientID            string
	HospPkNo             string
	PatientNo            string
	CardNo               string
	Name                 string
	Sex                  string
	Birthday             time.Time
	IDType               string
	IDNumber             string
	Phone                string
	Address              string
	FareIdentity         string
	MemberCardNo         string
	MemberLevel          string
	ContractUnit         string
	PatientCertificateNo string
	BirthPlace           string
	Ethnicity            string
	Nationality          string
	MaritalStatus        string
	OccupationCategory   string
	Occupation           string
	Education            string
	WorkUnit             string
	WorkUnitPhone        string
	WorkUnitPostcode     string
	HouseholdAddress     string
	HouseholdPhone       string
	HouseholdPostcode    string
	ContactName          string
	ContactAddress       string
	ContactPhone         string
	ContactRelation      string
	CardStatus           string
	Status               string
	CreatedAt            time.Time
	CreatedBy            string
	UpdatedAt            time.Time
	UpdatedBy            string
	IsDeleted            bool
}

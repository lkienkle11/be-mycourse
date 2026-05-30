package infra

import "mycourse-io-be/internal/shared/constants"

type profileDataRow struct {
	Headline          string `gorm:"size:255"`
	Bio               string `gorm:"type:text"`
	YearsOfExperience int
	CurrentJobTitle   string            `gorm:"size:255"`
	CurrentCompany    string            `gorm:"size:255"`
	CVFileID          string            `gorm:"size:64"`
	LinkedinURL       string            `gorm:"type:text"`
	GithubURL         string            `gorm:"type:text"`
	PortfolioLinks    *StringSliceJSON  `gorm:"type:jsonb"`
	Certificates      *CertificatesJSON `gorm:"type:jsonb"`
	IntroVideoFileID  string            `gorm:"size:64"`
}

type applicationRow struct {
	ID              uint   `gorm:"primaryKey"`
	UserID          uint   `gorm:"not null"`
	ReviewStatus    string `gorm:"size:32;not null"`
	RejectionReason string `gorm:"type:text"`
	profileDataRow
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

func (applicationRow) TableName() string { return constants.TableInstructorApplications }

type profileRow struct {
	ID     uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null"`
	profileDataRow
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

func (profileRow) TableName() string { return constants.TableInstructorProfiles }

type expertiseTopicRow struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	TopicID   uint
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

func (expertiseTopicRow) TableName() string { return constants.TableInstructorExpertiseTopics }

type expertiseSkillRow struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	SkillID   uint
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

func (expertiseSkillRow) TableName() string { return constants.TableInstructorExpertiseSkills }

type ticketRow struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint
	Subject   string `gorm:"size:255"`
	Status    string `gorm:"size:32"`
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

func (ticketRow) TableName() string { return constants.TableInstructorTickets }

type ticketMessageRow struct {
	ID           uint `gorm:"primaryKey"`
	TicketID     uint
	AuthorUserID uint
	Body         string `gorm:"type:text"`
	CreatedAt    int64
	UpdatedAt    int64
	DeletedAt    *int64
}

func (ticketMessageRow) TableName() string { return constants.TableInstructorTicketMessages }

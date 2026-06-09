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
	ID              string `gorm:"column:id;primaryKey;type:uuid"`
	UserID          string `gorm:"type:uuid;not null"`
	ReviewStatus    string `gorm:"size:32;not null"`
	RejectionReason string `gorm:"type:text"`
	profileDataRow
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

func (applicationRow) TableName() string { return constants.TableInstructorApplications }

type profileRow struct {
	ID     string `gorm:"column:id;primaryKey;type:uuid"`
	UserID string `gorm:"type:uuid;not null"`
	profileDataRow
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

func (profileRow) TableName() string { return constants.TableInstructorProfiles }

type expertiseTopicRow struct {
	ID        string `gorm:"column:id;primaryKey;type:uuid"`
	UserID    string `gorm:"column:user_id;type:uuid"`
	TopicID   string `gorm:"column:topic_id;type:uuid"`
	CreatedAt int64  `gorm:"column:created_at"`
	UpdatedAt int64  `gorm:"column:updated_at"`
	DeletedAt *int64 `gorm:"column:deleted_at"`
}

func (expertiseTopicRow) TableName() string { return constants.TableInstructorExpertiseTopics }

type expertiseSkillRow struct {
	ID        string `gorm:"column:id;primaryKey;type:uuid"`
	UserID    string `gorm:"column:user_id;type:uuid"`
	SkillID   string `gorm:"column:skill_id;type:uuid"`
	CreatedAt int64  `gorm:"column:created_at"`
	UpdatedAt int64  `gorm:"column:updated_at"`
	DeletedAt *int64 `gorm:"column:deleted_at"`
}

func (expertiseSkillRow) TableName() string { return constants.TableInstructorExpertiseSkills }

type ticketRow struct {
	ID        string `gorm:"column:id;primaryKey;type:uuid"`
	UserID    string `gorm:"type:uuid"`
	Subject   string `gorm:"size:255"`
	Status    string `gorm:"size:32"`
	CreatedAt int64
	UpdatedAt int64
	DeletedAt *int64
}

func (ticketRow) TableName() string { return constants.TableInstructorTickets }

type ticketMessageRow struct {
	ID           string `gorm:"column:id;primaryKey;type:uuid"`
	TicketID     string `gorm:"column:ticket_id;type:uuid"`
	AuthorUserID string `gorm:"type:uuid"`
	Body         string `gorm:"type:text"`
	CreatedAt    int64
	UpdatedAt    int64
	DeletedAt    *int64
}

func (ticketMessageRow) TableName() string { return constants.TableInstructorTicketMessages }

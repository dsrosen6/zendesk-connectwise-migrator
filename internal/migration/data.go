package migration

import (
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"strings"
	"time"
)

type Data struct {
	AllOrgs      map[string]*orgMigrationDetails
	UsersInPsa   map[string]*userMigrationDetails
	TicketsInPsa map[string]int

	PsaInfo        PsaInfo
	Tags           []tagDetails
	SelectedOrgs   []*orgMigrationDetails
	UsersToMigrate map[string]*userMigrationDetails

	CurrentMigratingOrg string
	Output              strings.Builder
}

func (c *Client) newData() *Data {
	return &Data{
		AllOrgs:        make(map[string]*orgMigrationDetails),
		UsersInPsa:     make(map[string]*userMigrationDetails),
		TicketsInPsa:   make(map[string]int),
		UsersToMigrate: make(map[string]*userMigrationDetails),
		PsaInfo: PsaInfo{
			Board:                  &psa.Board{Id: c.Cfg.Connectwise.DestinationBoardId},
			StatusOpen:             &psa.Status{Id: c.Cfg.Connectwise.OpenStatusId},
			StatusClosed:           &psa.Status{Id: c.Cfg.Connectwise.ClosedStatusId},
			ZendeskTicketIdField:   &psa.CustomField{Id: c.Cfg.Connectwise.FieldIds.ZendeskTicketId},
			ZendeskClosedDateField: &psa.CustomField{Id: c.Cfg.Connectwise.FieldIds.ZendeskClosedDate},
		},
	}
}

type orgMigrationDetails struct {
	ZendeskOrg *zendesk.Organization `json:"zendesk_org"`
	PsaOrg     *psa.Company          `json:"psa_org"`

	Tag        *tagDetails `json:"zendesk_tag"`
	HasTickets bool        `json:"has_tickets"`
	Migrated   bool        `json:"org_migrated"`

	MigrationSelected bool `json:"migration_selected"`
}

type tagDetails struct {
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type userMigrationDetails struct {
	ZendeskUser  *zendesk.User `json:"zendesk_user"`
	PsaContact   *psa.Contact  `json:"psa_contact"`
	PsaCompany   *psa.Company
	UserMigrated bool `json:"migrated"`

	HasTickets bool `json:"has_tickets"`
}

type PsaInfo struct {
	Board                  *psa.Board
	StatusOpen             *psa.Status
	StatusClosed           *psa.Status
	ZendeskTicketIdField   *psa.CustomField
	ZendeskClosedDateField *psa.CustomField
}

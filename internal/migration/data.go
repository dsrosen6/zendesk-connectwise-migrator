package migration

import (
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"strings"
)

type Data struct {
	AllOrgs      map[string]*orgMigrationDetails
	UsersInPsa   map[string]*userMigrationDetails
	TicketsInPsa map[string]int

	PsaInfo        PsaInfo
	Tags           []tagDetails
	SelectedOrgs   []*orgMigrationDetails
	UsersToMigrate map[string]*userMigrationDetails

	Output strings.Builder
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

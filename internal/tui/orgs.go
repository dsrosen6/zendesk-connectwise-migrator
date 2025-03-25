package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"time"
)

type orgMigrationModel struct {
	client *migration.Client
	data   *MigrationData
	tags   []tagDetails
	orgMigTotals
	status orgMigStatus
	done   bool
}

type orgMigrationDetails struct {
	ZendeskOrg *zendesk.Organization `json:"zendesk_org"`
	PsaOrg     *psa.Company          `json:"psa_org"`

	Tag         *tagDetails `json:"zendesk_tag"`
	HasTickets  bool        `json:"has_tickets"`
	OrgMigrated bool        `json:"org_migrated"`

	UserMigSelected bool                             `json:"user_migration_selected"`
	UsersToMigrate  map[string]*userMigrationDetails `json:"users_to_migrate"`
	UserMigDone     bool

	TicketMigSelected bool `json:"ticket_migration_selected"`
}

type tagDetails struct {
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type orgMigTotals struct {
	totalProcessed   int
	totalWithTickets int
	totalNotInPsa    int
	totalMigrated    int
	totalErrors      int
}

type orgMigStatus string

type switchOrgMigStatusMsg string

func switchOrgMigStatus(s orgMigStatus) tea.Cmd {
	return func() tea.Msg {
		return switchOrgMigStatusMsg(s)
	}
}

const (
	awaitingStart      orgMigStatus = "awaitingStart"
	gettingTags        orgMigStatus = "gettingTags"
	gettingZendeskOrgs orgMigStatus = "gettingZendeskOrgs"
	comparingOrgs      orgMigStatus = "comparingOrgs"
	orgMigDone         orgMigStatus = "orgMigDone"
)

func newOrgMigrationModel(mc *migration.Client, data *MigrationData) *orgMigrationModel {
	return &orgMigrationModel{
		client: mc,
		data:   data,
		status: awaitingStart,
	}
}

func (m *orgMigrationModel) Init() tea.Cmd {
	return nil
}

func (m *orgMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case switchOrgMigStatusMsg:
		switch msg {
		case switchOrgMigStatusMsg(gettingTags):

			m.status = gettingTags
			return m, tea.Batch(m.getTagDetails(), switchUserMigStatus(userMigWaitingForOrgs), switchTicketMigStatus(ticketMigWaitingForOrgs))

		case switchOrgMigStatusMsg(gettingZendeskOrgs):
			slog.Debug("org checker: got tags", "tags", m.tags)
			m.orgMigTotals = orgMigTotals{}
			m.status = gettingZendeskOrgs
			return m, m.getOrgs()

		case switchOrgMigStatusMsg(comparingOrgs):
			slog.Debug("org checker: got orgs", "total", len(m.data.Orgs))
			m.status = comparingOrgs
			var checkOrgCmds []tea.Cmd
			for _, org := range m.data.Orgs {
				checkOrgCmds = append(checkOrgCmds, m.checkOrg(org))
			}
			return m, tea.Sequence(checkOrgCmds...)

		case switchOrgMigStatusMsg(orgMigDone):
			slog.Debug("org checker: done checking orgs", "totalOrgs", len(m.data.Orgs))
			m.status = orgMigDone
			cmds = append(cmds, saveDataCmd())
		}
	}

	if m.status == comparingOrgs && !m.done {
		if len(m.data.Orgs) == m.totalProcessed {
			cmd = switchOrgMigStatus(orgMigDone)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *orgMigrationModel) View() string {
	var s string
	switch m.status {
	case awaitingStart:
		s = "Press SPACE to begin org migration"
	case gettingTags:
		s = runSpinner("Getting Zendesk tags")
	case gettingZendeskOrgs:
		s = runSpinner("Getting Zendesk orgs")
	case comparingOrgs:
		counterStatus := fmt.Sprintf("Checking for PSA orgs: %d/%d", m.totalProcessed, len(m.data.Orgs))
		s = runSpinner(counterStatus)
	case orgMigDone:
		s = fmt.Sprintf("Org migration done - press %s to run again\n\n%s", textNormalAdaptive("SPACE"), m.constructSummary())
	}

	return s
}

func (m *orgMigrationModel) getTagDetails() tea.Cmd {
	return func() tea.Msg {
		for _, tag := range m.client.Cfg.Zendesk.TagsToMigrate {
			tm := &timeConversionDetails{
				startString:   tag.StartDate,
				endString:     tag.EndDate,
				startFallback: m.client.Cfg.Zendesk.MasterStartDate,
				endFallback:   m.client.Cfg.Zendesk.MasterEndDate,
			}

			start, end, err := convertDateStringsToTimeTime(tm)
			if err != nil {
				return timeConvertErrMsg{err}
			}

			td := tagDetails{
				Name:      tag.Name,
				StartDate: start,
				EndDate:   end,
			}

			m.tags = append(m.tags, td)
		}
		return switchOrgMigStatusMsg(gettingZendeskOrgs)
	}
}

func (m *orgMigrationModel) getOrgs() tea.Cmd {
	return func() tea.Msg {
		slog.Info("getting orgs for tags", "tags", m.client.Cfg.Zendesk.TagsToMigrate)
		for _, tag := range m.tags {
			slog.Debug("getting orgs for tag", "tag", tag.Name)
			q := &zendesk.SearchQuery{}
			q.Tags = []string{tag.Name}

			slog.Info("getting all orgs from zendesk for tag group", "tag", tag.Name)

			orgs, err := m.client.ZendeskClient.GetOrganizationsWithQuery(ctx, *q)
			if err != nil {
				return apiErrMsg{err} // TODO: actually error handle, don't just leave
			}

			for _, org := range orgs {
				idString := fmt.Sprintf("%d", org.Id)
				if _, ok := m.data.Orgs[idString]; !ok {
					slog.Debug("adding org to migration data", "zendeskOrgId", idString, "orgName", org.Name)

					md := &orgMigrationDetails{
						ZendeskOrg:     &org,
						Tag:            &tag,
						UsersToMigrate: make(map[string]*userMigrationDetails),
					}

					m.data.addOrgToOrgsMap(idString, md)
				} else {
					slog.Debug("org already in migration data", "zendeskOrgId", org.Id, "orgName", org.Name)
				}
			}
		}

		return switchOrgMigStatusMsg(comparingOrgs)
	}
}

func (d *MigrationData) addOrgToOrgsMap(idString string, org *orgMigrationDetails) {
	d.Orgs[idString] = org
}

func (m *orgMigrationModel) checkOrg(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		if org.OrgMigrated {
			slog.Debug("org already migrated",
				"orgName", org.ZendeskOrg.Name,
				"psaCompanyId", org.PsaOrg.Id,
				"zendeskPsaCompanyField", org.ZendeskOrg.OrganizationFields.PSACompanyId,
			)
			m.data.writeToOutput(goodBlueOutput("NO ACTION", fmt.Sprintf("Org already migrated: %s", org.ZendeskOrg.Name)))
			m.totalProcessed++
			m.totalMigrated++
			m.totalWithTickets++
			return nil
		}

		q := zendesk.SearchQuery{
			TicketsOrganizationId: org.ZendeskOrg.Id,
			TicketCreatedAfter:    org.Tag.StartDate,
			TicketCreatedBefore:   org.Tag.EndDate,
		}

		q.TicketsOrganizationId = org.ZendeskOrg.Id

		tickets, err := m.client.ZendeskClient.GetTicketsWithQuery(ctx, q, 20, 1)
		if err != nil {
			slog.Error("getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't get tickets for org %s: %w", org.ZendeskOrg.Name, err)))
			m.totalProcessed++
			m.totalErrors++
			return nil
		}

		if len(tickets) > 0 {
			org.HasTickets = true
			m.totalWithTickets++
		} else {
			// We only care about orgs with tickets - no need to check further
			slog.Debug("org has no tickets", "orgName", org.ZendeskOrg.Name)
			m.data.writeToOutput(infoOutput("INFO", fmt.Sprintf("org has no tickets: %s", org.ZendeskOrg.Name)))
			m.totalProcessed++
			return nil
		}

		org.PsaOrg, err = m.matchZdOrgToCwCompany(org.ZendeskOrg)
		if err != nil {
			// TODO: Add actual org creation
			slog.Warn("org is not in PSA", "orgName", org.ZendeskOrg.Name)
			m.data.writeToOutput(warnYellowOutput("WARNING", fmt.Sprintf("Error: org not in PSA: %s", org.ZendeskOrg.Name)))
			m.totalProcessed++
			m.totalNotInPsa++
			return nil
		}

		if err := m.updateCompanyFieldValue(org); err != nil {
			slog.Error("updating company field value in zendesk", "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't zendesk company field value for org %s: %w", org.ZendeskOrg.Name, err)))
			m.totalProcessed++
			m.totalErrors++
			return nil
		}

		if org.PsaOrg != nil && org.ZendeskOrg.OrganizationFields.PSACompanyId == int64(org.PsaOrg.Id) {
			org.OrgMigrated = true
			slog.Debug("org is ready for user migration", "orgName", org.ZendeskOrg.Name)
			m.data.writeToOutput(goodBlueOutput("NO ACTION", fmt.Sprintf("Org is ready for migration: %s", org.ZendeskOrg.Name)))
			m.totalProcessed++
			m.totalMigrated++
		}

		return nil
	}
}

func (m *orgMigrationModel) updateCompanyFieldValue(org *orgMigrationDetails) error {
	if org.ZendeskOrg.OrganizationFields.PSACompanyId == int64(org.PsaOrg.Id) {
		slog.Debug("zendesk org already has PSA company id field", "orgName", org.ZendeskOrg.Name, "psaCompanyId", org.ZendeskOrg.OrganizationFields.PSACompanyId)
		return nil
	}

	if org.PsaOrg.Id != 0 {
		org.ZendeskOrg.OrganizationFields.PSACompanyId = int64(org.PsaOrg.Id)

		var err error
		org.ZendeskOrg, err = m.client.ZendeskClient.UpdateOrganization(ctx, org.ZendeskOrg)
		if err != nil {
			return fmt.Errorf("updating organization with PSA company id: %w", err)
		}

		slog.Info("updated zendesk organization with PSA company id", "orgName", org.ZendeskOrg.Name, "psaCompanyId", org.PsaOrg.Id)
		return nil
	} else {
		slog.Error("org psa id is 0 - cannot update psa_company field in zendesk", "orgName", org.ZendeskOrg.Name)
		return errors.New("org psa id is 0 - cannot update psa_company field in zendesk")
	}
}

func (m *orgMigrationModel) matchZdOrgToCwCompany(org *zendesk.Organization) (*psa.Company, error) {
	comp, err := m.client.CwClient.GetCompanyByName(ctx, org.Name)
	if err != nil {
		return nil, err
	}

	return comp, nil
}

func (m *orgMigrationModel) constructSummary() string {
	return fmt.Sprintf(
		"%s %d\n"+
			"%s %d\n"+
			"%s %d/%d\n"+
			"%s %d\n\n",
		textNormalAdaptive("Orgs With Tickets:"), m.totalWithTickets,
		textNormalAdaptive("Not in PSA:"), m.totalNotInPsa,
		textNormalAdaptive("Ready for User Migration:"), m.totalMigrated, m.totalWithTickets,
		textNormalAdaptive("Errored"), m.totalErrors)
}

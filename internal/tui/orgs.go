package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/migration"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"sort"
	"time"
)

type orgMigrationDetails struct {
	ZendeskOrg *zendesk.Organization `json:"zendesk_org"`
	PsaOrg     *psa.Company          `json:"psa_org"`

	Tag         *tagDetails `json:"zendesk_tag"`
	HasTickets  bool        `json:"has_tickets"`
	OrgMigrated bool        `json:"org_migrated"`

	MigrationSelected bool `json:"migration_selected"`
	UserMigDone       bool
}

type tagDetails struct {
	Name      string    `json:"name"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

func (m *RootModel) getTagDetails() tea.Cmd {
	return func() tea.Msg {
		for _, tag := range m.client.Cfg.Zendesk.TagsToMigrate {
			slog.Debug("getting tag details", "tag", tag.Name)
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

			m.data.Tags = append(m.data.Tags, td)
		}

		return switchStatusMsg(gettingZendeskOrgs)
	}
}

func (m *RootModel) getOrgs() tea.Cmd {
	return func() tea.Msg {
		slog.Info("getting orgs for tags", "tags", m.client.Cfg.Zendesk.TagsToMigrate)
		for _, tag := range m.data.Tags {
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
						ZendeskOrg: &org,
						Tag:        &tag,
					}

					m.data.addOrgToOrgsMap(idString, md)
				} else {
					slog.Debug("org already in migration data", "zendeskOrgId", org.Id, "orgName", org.Name)
				}
			}
		}

		return switchStatusMsg(comparingOrgs)
	}
}

func (d *MigrationData) addOrgToOrgsMap(idString string, org *orgMigrationDetails) {
	d.Orgs[idString] = org
}

func (m *RootModel) checkOrg(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		if org.OrgMigrated {
			slog.Debug("org already migrated",
				"orgName", org.ZendeskOrg.Name,
				"psaCompanyId", org.PsaOrg.Id,
				"zendeskPsaCompanyField", org.ZendeskOrg.OrganizationFields.PSACompanyId,
			)
			m.data.writeToOutput(goodBlueOutput("NO ACTION", fmt.Sprintf("Org already migrated: %s", org.ZendeskOrg.Name)))
			m.orgsProcessed++
			m.orgsMigrated++
			m.orgsWithTickets++
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
			m.orgsProcessed++
			m.totalErrors++
			return nil
		}

		if len(tickets) > 0 {
			org.HasTickets = true
			m.orgsWithTickets++
		} else {
			// We only care about orgs with tickets - no need to check further
			slog.Debug("org has no tickets", "orgName", org.ZendeskOrg.Name)
			m.data.writeToOutput(infoOutput("INFO", fmt.Sprintf("org has no tickets: %s", org.ZendeskOrg.Name)))
			m.orgsProcessed++
			return nil
		}

		org.PsaOrg, err = m.matchZdOrgToCwCompany(org.ZendeskOrg)
		if err != nil {
			// TODO: Add actual org creation
			slog.Warn("org is not in PSA", "orgName", org.ZendeskOrg.Name)
			m.data.writeToOutput(warnYellowOutput("WARNING", fmt.Sprintf("Error: org not in PSA: %s", org.ZendeskOrg.Name)))
			m.orgsProcessed++
			m.orgsNotInPsa++
			return nil
		}

		if err := m.updateCompanyFieldValue(org); err != nil {
			slog.Error("updating company field value in zendesk", "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't zendesk company field value for org %s: %w", org.ZendeskOrg.Name, err)))
			m.orgsProcessed++
			m.totalErrors++
			return nil
		}

		if org.PsaOrg != nil && org.ZendeskOrg.OrganizationFields.PSACompanyId == int64(org.PsaOrg.Id) {
			org.OrgMigrated = true
			slog.Debug("org is ready for user migration", "orgName", org.ZendeskOrg.Name)
			m.data.writeToOutput(goodBlueOutput("NO ACTION", fmt.Sprintf("Org is ready for migration: %s", org.ZendeskOrg.Name)))
			m.orgsProcessed++
			m.orgsMigrated++
		}

		return nil
	}
}

func (m *RootModel) updateCompanyFieldValue(org *orgMigrationDetails) error {
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

func (m *RootModel) matchZdOrgToCwCompany(org *zendesk.Organization) (*psa.Company, error) {
	comp, err := m.client.CwClient.GetCompanyByName(ctx, org.Name)
	if err != nil {
		return nil, err
	}

	return comp, nil
}

func (m *RootModel) orgSelectionForm() *huh.Form {
	return huh.NewForm(

		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Migrate all confirmed orgs?").
				Description("If not, select the organizations you want to migrate on the next screen.").
				Options(
					huh.NewOption("All Orgs", true),
					huh.NewOption("Select Orgs", false)).
				Value(&m.allOrgsSelected),
		),
		huh.NewGroup(
			huh.NewMultiSelect[*orgMigrationDetails]().
				Title("Pick the orgs you'd like to migrate users for").
				Description("Use Space to select, and Enter/Return to submit").
				Options(m.orgOptions()...).
				Value(&m.data.SelectedOrgs),
		).WithHideFunc(func() bool { return m.allOrgsSelected == true }),
	).WithHeight(verticalLeftForMainView).WithShowHelp(false).WithTheme(migration.CustomHuhTheme())
}

func (m *RootModel) orgOptions() []huh.Option[*orgMigrationDetails] {
	var orgOptions []huh.Option[*orgMigrationDetails]
	for _, org := range m.data.Orgs {
		if org.OrgMigrated {
			opt := huh.Option[*orgMigrationDetails]{
				Key:   org.ZendeskOrg.Name,
				Value: org,
			}

			orgOptions = append(orgOptions, opt)
		}
	}

	sort.Slice(orgOptions, func(i, j int) bool {
		return orgOptions[i].Key < orgOptions[j].Key
	})

	return orgOptions
}

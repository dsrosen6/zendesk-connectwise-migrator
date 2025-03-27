package migration

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"sort"
	"time"
)

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

func (m *Model) getTagDetails() tea.Cmd {
	return func() tea.Msg {
		m.data.Tags = []tagDetails{}
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

func (m *Model) getOrgs() tea.Cmd {
	return func() tea.Msg {
		slog.Debug("getting orgs for tags", "tags", m.client.Cfg.Zendesk.TagsToMigrate)
		for _, tag := range m.data.Tags {
			slog.Debug("getting orgs for tag", "tag", tag.Name)
			q := &zendesk.SearchQuery{}
			q.Tags = []string{tag.Name}

			slog.Debug("getting all orgs from zendesk for tag group", "tag", tag.Name)

			orgs, err := m.client.ZendeskClient.GetOrganizationsWithQuery(m.ctx, *q)
			if err != nil {
				return apiErrMsg{err} // TODO: actually error handle, don't just leave
			}

			for _, org := range orgs {
				idString := fmt.Sprintf("%d", org.Id)
				if _, ok := m.data.AllOrgs[idString]; !ok {
					slog.Debug("adding org to migration data", "zendeskOrgId", idString, "orgName", org.Name)

					md := &orgMigrationDetails{
						ZendeskOrg: &org,
						Tag:        &tag,
					}

					m.data.AllOrgs[idString] = md
				} else {
					slog.Debug("org already in migration data", "zendeskOrgId", org.Id, "orgName", org.Name)
				}
			}
		}

		return switchStatusMsg(comparingOrgs)
	}
}

func (m *Model) checkOrg(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		if org.Migrated {
			slog.Debug("org already migrated", "orgName", org.ZendeskOrg.Name)
			m.orgsChecked++
			m.orgsMigrated++
			return nil
		}

		q := zendesk.SearchQuery{
			TicketsOrganizationId: org.ZendeskOrg.Id,
			TicketCreatedAfter:    org.Tag.StartDate,
			TicketCreatedBefore:   org.Tag.EndDate,
		}

		if m.client.Cfg.MigrateOpenTickets {
			q.GetOpenTickets = true
		}

		tickets, err := m.client.ZendeskClient.GetTicketsWithQuery(m.ctx, q, 20, 1)
		if err != nil {
			slog.Error("getting tickets for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't get tickets for org %s: %w", org.ZendeskOrg.Name, err)), errOutput)
			m.orgsChecked++
			m.totalErrors++
			return nil
		}

		if len(tickets) == 0 {
			// We only care about orgs with tickets - no need to check further
			slog.Debug("org has no tickets", "orgName", org.ZendeskOrg.Name)
			m.orgsChecked++
			return nil
		}

		org.PsaOrg, err = m.matchZdOrgToCwCompany(org.ZendeskOrg)
		if err != nil {
			slog.Warn("org is not in PSA", "orgName", org.ZendeskOrg.Name)
			m.writeToOutput(warnYellowOutput("WARNING", fmt.Sprintf("org not in PSA: %s", org.ZendeskOrg.Name)), warnOutput)
			m.orgsChecked++
			m.orgsNotInPsa++
			return nil
		}

		if err := m.updateCompanyFieldValue(org); err != nil {
			slog.Error("updating company field value in zendesk", "orgName", org.ZendeskOrg.Name, "zendeskOrgId", org.ZendeskOrg.Id, "error", err)
			m.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't zendesk company field value for org %s: %w", org.ZendeskOrg.Name, err)), errOutput)
			m.orgsChecked++
			m.totalErrors++
			return nil
		}

		if org.PsaOrg != nil && org.ZendeskOrg.OrganizationFields.PSACompanyId == int64(org.PsaOrg.Id) {
			slog.Info("org ready for migration", "orgName", org.ZendeskOrg.Name, "zendeskOrgId", org.ZendeskOrg.Id)
			m.writeToOutput(goodBlueOutput("NO ACTION", fmt.Sprintf("Org is ready for migration: %s", org.ZendeskOrg.Name)), noActionOutput)
			m.orgsChecked++
			m.orgsMigrated++
			org.Migrated = true
		}

		return nil
	}
}

func (m *Model) updateCompanyFieldValue(org *orgMigrationDetails) error {
	if org.ZendeskOrg.OrganizationFields.PSACompanyId == int64(org.PsaOrg.Id) {
		slog.Debug("zendesk org already has PSA company id field", "orgName", org.ZendeskOrg.Name, "zendeskOrgId", org.ZendeskOrg.Id, "psaCompanyId", org.ZendeskOrg.OrganizationFields.PSACompanyId)
		return nil
	}

	if org.PsaOrg.Id != 0 {
		org.ZendeskOrg.OrganizationFields.PSACompanyId = int64(org.PsaOrg.Id)

		var err error
		org.ZendeskOrg, err = m.client.ZendeskClient.UpdateOrganization(m.ctx, org.ZendeskOrg)
		if err != nil {
			return fmt.Errorf("updating organization with PSA company id: %w", err)
		}

		slog.Info("updated zendesk organization with PSA company id", "orgName", org.ZendeskOrg.Name, "zendeskOrgId", org.ZendeskOrg.Id, "psaCompanyId", org.PsaOrg.Id)
		return nil
	} else {
		slog.Error("org psa id is 0 - cannot update psa_company field in zendesk", "orgName", org.ZendeskOrg.Name, "zendeskOrgId", org.ZendeskOrg.Id)
		return errors.New("org psa id is 0 - cannot update psa_company field in zendesk")
	}
}

func (m *Model) matchZdOrgToCwCompany(org *zendesk.Organization) (*psa.Company, error) {
	comp, err := m.client.CwClient.GetCompanyByName(m.ctx, org.Name)
	if err != nil {
		return nil, err
	}

	return comp, nil
}

func (m *Model) orgSelectionForm() *huh.Form {
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
	).WithHeight(verticalLeftForMainView).WithShowHelp(false).WithTheme(customFormTheme())
}

func (m *Model) orgOptions() []huh.Option[*orgMigrationDetails] {
	var orgOptions []huh.Option[*orgMigrationDetails]
	for _, org := range m.data.AllOrgs {
		if org.Migrated {
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

func convertDateStringsToTimeTime(details *timeConversionDetails) (time.Time, time.Time, error) {
	var startDate, endDate time.Time
	var err error

	start := details.startString
	if start == "" {
		start = details.startFallback
	}

	end := details.endString
	if end == "" {
		end = details.endFallback
	}

	if start != "" {
		startDate, err = convertStrTime(start)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("converting start date string to time.Time: %w", err)
		}
	}

	if end != "" {
		endDate, err = convertStrTime(end)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("converting end date string to time.Time: %w", err)
		}
	}

	return startDate, endDate, nil
}

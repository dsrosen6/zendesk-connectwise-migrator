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
	"strings"
)

type userMigrationModel struct {
	client         *migration.Client
	data           *migrationData
	form           *huh.Form
	formHeight     int
	selectedOrgs   []*orgMigrationDetails
	allOrgsChecked bool
	totalToMigrate int
	checkedTotal   int
	status         userMigStatus
	done           bool
}

type userMigStatus string
type switchUserMigStatusMsg string

func switchUserMigStatus(s userMigStatus) tea.Cmd {
	return func() tea.Msg {
		return switchUserMigStatusMsg(s)
	}
}

const (
	noOrgs         userMigStatus = "noOrgs"
	pickingOrgs    userMigStatus = "pickingOrgs"
	gettingUsers   userMigStatus = "gettingUsers"
	migratingUsers userMigStatus = "migratingUsers"
	userMigDone    userMigStatus = "userMigDone"
)

func newUserMigrationModel(mc *migration.Client, data *migrationData) *userMigrationModel {
	m := &userMigrationModel{
		client: mc,
		data:   data,
		status: noOrgs,
	}

	return m
}

func (m *userMigrationModel) Init() tea.Cmd {
	return nil
}

func (m *userMigrationModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case calculateDimensionsMsg:
		if m.form != nil {
			m.form.WithHeight(verticalLeftForMainView)
		}

	case switchUserMigStatusMsg:
		slog.Debug("user migration: got switchUserMigStatusMsg", "status", msg)
		switch msg {
		case switchUserMigStatusMsg(pickingOrgs):
			slog.Debug("got pickingOrgs status")
			m.form = m.orgSelectionForm()
			m.status = pickingOrgs
			cmds = append(cmds, m.form.Init())
			return m, tea.Batch(cmds...)

		case switchUserMigStatusMsg(gettingUsers):
			m.totalToMigrate = 0
			m.status = gettingUsers
			for _, org := range m.data.orgs {
				if org.readyUsers && org.userMigSelected {
					m.totalToMigrate++
					cmds = append(cmds, m.getUsersToMigrate(org))
				}
			}

			slog.Debug("user migration: orgs picked", "totalOrgs", m.totalToMigrate)
			return m, tea.Batch(cmds...)
		case switchUserMigStatusMsg(migratingUsers):
			m.status = migratingUsers
			for _, org := range m.data.orgs {
				if len(org.usersToMigrate) > 0 {
					for _, user := range org.usersToMigrate {
						cmds = append(cmds, m.migrateUser(org, user))
					}
				}
			}
			return m, tea.Sequence(cmds...)
		}
	}

	if len(m.data.orgs) == 0 {
		m.status = noOrgs
	}

	if m.status == pickingOrgs {
		form, cmd := m.form.Update(msg)
		cmds = append(cmds, cmd)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}

		// Once the form is submitted, mark all selected orgs as needing migration
		if m.form.State == huh.StateCompleted {

			if m.allOrgsChecked {
				for _, org := range m.data.orgs {
					if !org.readyUsers {
						continue
					}
					org.userMigSelected = true
				}
			} else {
				for _, org := range m.selectedOrgs {
					org.userMigSelected = true
				}
			}

			cmds = append(cmds, switchUserMigStatus(gettingUsers))
		}
	}

	if m.status == gettingUsers {
		cmds = append(cmds, switchUserMigStatus(migratingUsers))
	}

	return m, tea.Batch(cmds...)
}

func (m *userMigrationModel) View() string {
	var s string
	var showDetails bool
	switch m.status {
	case noOrgs:
		s = "No orgs have been loaded! Please return to the main menu and select Organizations, then return."
	case pickingOrgs:
		s = m.form.View()
	case gettingUsers:
		s = runSpinner("Getting users")
		showDetails = true
	case migratingUsers:
		s = runSpinner("Migrating users")
		showDetails = true
	}

	if showDetails {
		s += fmt.Sprintf("\n\nProcessed: %d/%d\n", m.checkedTotal, m.totalToMigrate)
	}

	return s
}

func (m *userMigrationModel) orgSelectionForm() *huh.Form {
	return huh.NewForm(

		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Migrate all confirmed orgs?").
				Description("If not, select the organizations you want to migrate on the next screen.").
				Options(
					huh.NewOption("All Orgs", true),
					huh.NewOption("Select Orgs", false)).
				Value(&m.allOrgsChecked),
		),
		huh.NewGroup(
			huh.NewMultiSelect[*orgMigrationDetails]().
				Title("Pick the orgs you'd like to migrate users for").
				Description("Use Space to select, and Enter/Return to submit").
				Options(m.orgOptions()...).
				Value(&m.selectedOrgs),
		).WithHideFunc(func() bool { return m.allOrgsChecked == true }),
	).WithHeight(verticalLeftForMainView).WithShowHelp(false).WithTheme(migration.CustomHuhTheme())
}

func (m *userMigrationModel) orgOptions() []huh.Option[*orgMigrationDetails] {
	var orgOptions []huh.Option[*orgMigrationDetails]
	for _, org := range m.data.orgs {
		if org.readyUsers {
			opt := huh.Option[*orgMigrationDetails]{
				Key:   org.zendeskOrg.Name,
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

func (m *userMigrationModel) getUsersToMigrate(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		users, err := m.client.ZendeskClient.GetOrganizationUsers(ctx, org.zendeskOrg.Id)
		if err != nil {
			slog.Error("getting users for org", "orgName", org.zendeskOrg.Name, "error", err)
			org.userMigErrors = append(org.userMigErrors, fmt.Errorf("getting users: %w", err))
			return nil
		}

		for _, user := range users {
			slog.Debug("got user", "orgName", org.zendeskOrg.Name, "userName", user.Name)
			org.usersToMigrate = append(org.usersToMigrate, &userMigrationDetails{zendeskUser: &user, psaCompany: org.psaOrg})
		}

		slog.Info("got users for org", "orgName", org.zendeskOrg.Name, "totalUsers", len(org.usersToMigrate))
		return nil
	}
}

func (m *userMigrationModel) migrateUser(org *orgMigrationDetails, user *userMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		var err error
		user.psaContact, err = m.matchZdUserToCwContact(user.zendeskUser)
		if err != nil {
			slog.Debug("user does not exist in psa - attempting to create new user", "userEmail", user.zendeskUser.Email)
			user.psaContact, err = m.createPsaContact(user)
			if err != nil {
				slog.Error("creating user", "userName", user.zendeskUser.Email, "error", err)
				org.userMigErrors = append(org.userMigErrors, fmt.Errorf("creating user %s: %w", user.zendeskUser.Email, err))
				return nil
			}
		}

		if err := m.updateContactFieldValue(user); err != nil {
			slog.Error("updating user contact field value in zendesk", "userEmail", user.zendeskUser.Email, "error", err)
			org.userMigErrors = append(org.userMigErrors, fmt.Errorf("updating contact field in zendesk for %s: %w", user.zendeskUser.Email, err))
			return nil
		}

		if user.psaContact != nil && user.zendeskUser.UserFields.PSAContactId == user.psaContact.Id {
			user.migrated = true
			slog.Debug("user is fully migrated", "userEmail", user.zendeskUser.Email, "psaContactId", user.psaContact.Id)
		}

		m.checkedTotal++
		return nil
	}
}

func (m *userMigrationModel) matchZdUserToCwContact(user *zendesk.User) (*psa.Contact, error) {
	contact, err := m.client.CwClient.GetContactByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	return contact, nil
}

func (m *userMigrationModel) createPsaContact(user *userMigrationDetails) (*psa.Contact, error) {
	c := &psa.Contact{}
	c.FirstName, c.LastName = separateName(user.zendeskUser.Name)

	if user.psaCompany == nil {
		return nil, errors.New("user psa company is nil")
	}
	c.Company = *user.psaCompany

	c.CommunicationItems = []psa.CommunicationItem{
		{
			Type:              psa.CommunicationItemType{Name: "Email"},
			Value:             user.zendeskUser.Email,
			DefaultFlag:       true,
			CommunicationType: "Email",
		},
	}

	return m.client.CwClient.PostContact(ctx, c)
}

func (m *userMigrationModel) updateContactFieldValue(user *userMigrationDetails) error {
	if user.zendeskUser.UserFields.PSAContactId == user.psaContact.Id {
		slog.Debug("zendesk user already has PSA contact id field",
			"userEmail", user.zendeskUser.Email,
			"psaContactId", user.psaContact.Id,
		)
	}

	if user.psaContact.Id != 0 {
		user.zendeskUser.UserFields.PSAContactId = user.psaContact.Id

		var err error
		user.zendeskUser, err = m.client.ZendeskClient.UpdateUser(ctx, user.zendeskUser)
		if err != nil {
			return fmt.Errorf("updating user with PSA contact id: %w", err)
		}

		slog.Info("updated zendesk user with PSA contact id", "userEmail", user.zendeskUser.Email)
		return nil
	} else {
		slog.Error("user psa id is 0 - cannot update psa_contact field in zendesk", "userName", user.zendeskUser.Name)
		return errors.New("user psa id is 0 - cannot update psa_contact field in zendesk")
	}
}

func separateName(name string) (string, string) {
	nameParts := strings.Split(name, " ")
	if len(nameParts) == 1 {
		return nameParts[0], ""
	}

	return nameParts[0], strings.Join(nameParts[1:], " ")
}

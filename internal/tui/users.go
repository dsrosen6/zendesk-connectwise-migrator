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
	client                  *migration.Client
	data                    *MigrationData
	form                    *huh.Form
	formHeight              int
	selectedOrgs            []*orgMigrationDetails
	allOrgsChecked          bool
	totalOrgsToMigrateUsers int
	totalOrgsChecked        int
	totalUsersToProcess     int
	totalNewUsersCreated    int
	totalUsersProcessed     int
	totalUsersSkipped       int
	totalOrgsDone           int

	status userMigStatus
	done   bool
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

func newUserMigrationModel(mc *migration.Client, data *MigrationData) *userMigrationModel {
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
		slog.Debug("user migration received calculate dimensions msg", "formHeight", verticalLeftForMainView)
		if m.form != nil {
			m.form.WithHeight(verticalLeftForMainView)
		}

	case switchUserMigStatusMsg:
		slog.Debug("user migration: got switchUserMigStatusMsg", "status", msg)
		switch msg {
		case switchUserMigStatusMsg(pickingOrgs):
			// Step 1
			slog.Debug("got pickingOrgs status")
			m.form = m.orgSelectionForm()
			m.status = pickingOrgs
			cmds = append(cmds, m.form.Init())
			return m, tea.Sequence(cmds...) // TODO: Switch to Batch when ready for speed

		case switchUserMigStatusMsg(gettingUsers):
			m.totalOrgsToMigrateUsers = 0
			m.totalOrgsChecked = 0
			m.totalUsersSkipped = 0
			m.status = gettingUsers
			for _, org := range m.data.Orgs {
				if org.ReadyUsers && org.UserMigSelected {
					m.totalOrgsToMigrateUsers++
					cmds = append(cmds, m.getUsersToMigrate(org))
				}
			}

			slog.Debug("user migration: orgs picked", "totalOrgs", m.totalOrgsToMigrateUsers)
			return m, tea.Sequence(cmds...) // TODO: Switch to Batch when ready for speed

		case switchUserMigStatusMsg(migratingUsers):
			m.status = migratingUsers
			for _, org := range m.data.Orgs {
				m.totalUsersToProcess += len(org.UsersToMigrate)
				cmds = append(cmds, m.migrateOrgUsers(org))
			}

			return m, tea.Sequence(cmds...)
		}
	}

	if len(m.data.Orgs) == 0 {
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
				for _, org := range m.data.Orgs {
					if !org.ReadyUsers {
						continue
					}
					org.UserMigSelected = true
				}
			} else {
				for _, org := range m.selectedOrgs {
					org.UserMigSelected = true
				}
			}

			cmds = append(cmds, switchUserMigStatus(gettingUsers))
		}
	}

	if m.status == gettingUsers && m.totalOrgsToMigrateUsers == m.totalOrgsChecked {
		slog.Debug("all orgs have been checked")
		cmds = append(cmds, switchUserMigStatus(migratingUsers))
	}

	if m.status == migratingUsers && m.totalUsersToProcess == m.totalUsersProcessed {
		m.status = userMigDone
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
	case userMigDone:
		s = "User migration done"
		showDetails = true
	}

	if showDetails {
		s += fmt.Sprintf("\n\nProcessed Orgs: %d/%d\n"+
			"Users Processed: %d/%d\n"+
			"New Users Created: %d\n"+
			"Users Skipped: %d\n",
			m.totalOrgsChecked, m.totalOrgsToMigrateUsers,
			m.totalUsersProcessed, m.totalUsersToProcess,
			m.totalNewUsersCreated,
			m.totalUsersSkipped)
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
	for _, org := range m.data.Orgs {
		if org.ReadyUsers {
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

func (m *userMigrationModel) getUsersToMigrate(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		users, err := m.client.ZendeskClient.GetOrganizationUsers(ctx, org.ZendeskOrg.Id)
		if err != nil {
			m.totalOrgsChecked++
			slog.Error("getting users for org", "orgName", org.ZendeskOrg.Name, "error", err)
			org.UserMigErrors = append(org.UserMigErrors, fmt.Errorf("getting users: %w", err))
			return nil
		}

		for _, user := range users {
			slog.Debug("got user", "orgName", org.ZendeskOrg.Name, "userName", user.Name)
			org.UsersToMigrate = append(org.UsersToMigrate, &userMigrationDetails{ZendeskUser: &user, PsaCompany: org.PsaOrg})
		}

		m.totalOrgsChecked++
		slog.Info("got users for org", "orgName", org.ZendeskOrg.Name, "totalUsers", len(org.UsersToMigrate))
		return nil
	}
}

func (m *userMigrationModel) migrateOrgUsers(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		for _, user := range org.UsersToMigrate {
			m.totalUsersProcessed++
			if user.ZendeskUser.Email == "" {
				slog.Warn("zendesk user has no email address - skipping", "userName", user.ZendeskUser.Name)
				m.totalUsersSkipped++
				continue
			}

			var err error
			user.PsaContact, err = m.matchZdUserToCwContact(user.ZendeskUser)
			if err != nil {

				if errors.Is(err, psa.NoUserFoundErr{}) {
					slog.Debug("user does not exist in psa - attempting to create new user", "userEmail", user.ZendeskUser.Email)
					user.PsaContact, err = m.createPsaContact(user)

					if err != nil {
						slog.Error("creating user", "userName", user.ZendeskUser.Email, "error", err)
						org.UserMigErrors = append(org.UserMigErrors, fmt.Errorf("creating user %s: %w", user.ZendeskUser.Email, err))
						continue
					}

					slog.Debug("matched zendesk user to psa user", "userEmail", user.ZendeskUser.Email)
					m.totalNewUsersCreated++

				} else {
					slog.Error("matching zendesk user to psa user", "userEmail", user.ZendeskUser.Email, "error", err)
					org.UserMigErrors = append(org.UserMigErrors, fmt.Errorf("matching zendesk user to psa user %s: %w", user.ZendeskUser.Email, err))
					continue
				}
			}

			if err := m.updateContactFieldValue(user); err != nil {
				slog.Error("updating user contact field value in zendesk", "userEmail", user.ZendeskUser.Email, "error", err)
				org.UserMigErrors = append(org.UserMigErrors, fmt.Errorf("updating contact field in zendesk for %s: %w", user.ZendeskUser.Email, err))
				continue
			}

			if user.PsaContact != nil && user.ZendeskUser.UserFields.PSAContactId == user.PsaContact.Id {
				user.Migrated = true
				slog.Debug("user is fully migrated", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
			}
		}
		m.totalOrgsDone++
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
	c := &psa.ContactPostBody{}
	c.FirstName, c.LastName = separateName(user.ZendeskUser.Name)

	if user.PsaCompany == nil {
		return nil, errors.New("user psa company is nil")
	}

	c.Company.Id = user.PsaCompany.Id

	c.CommunicationItems = []psa.CommunicationItem{
		{
			Type:              psa.CommunicationItemType{Name: "Email"},
			Value:             user.ZendeskUser.Email,
			DefaultFlag:       true,
			CommunicationType: "Email",
		},
	}

	return m.client.CwClient.PostContact(ctx, c)
}

func (m *userMigrationModel) updateContactFieldValue(user *userMigrationDetails) error {
	if user.ZendeskUser.UserFields.PSAContactId == user.PsaContact.Id {
		slog.Debug("zendesk user already has PSA contact id field",
			"userEmail", user.ZendeskUser.Email,
			"psaContactId", user.PsaContact.Id,
		)
		return nil
	}

	if user.PsaContact.Id != 0 {
		user.ZendeskUser.UserFields.PSAContactId = user.PsaContact.Id

		var err error
		user.ZendeskUser, err = m.client.ZendeskClient.UpdateUser(ctx, user.ZendeskUser)
		if err != nil {
			return fmt.Errorf("updating user with PSA contact id: %w", err)
		}

		slog.Info("updated zendesk user with PSA contact id", "userEmail", user.ZendeskUser.Email)
		return nil
	} else {
		slog.Error("user psa id is 0 - cannot update psa_contact field in zendesk", "userName", user.ZendeskUser.Name)
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

package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/psa"
	"github.com/dsrosen/zendesk-connectwise-migrator/internal/zendesk"
	"log/slog"
	"strings"
)

type userMigrationDetails struct {
	ZendeskUser  *zendesk.User `json:"zendesk_user"`
	PsaContact   *psa.Contact  `json:"psa_contact"`
	PsaCompany   *psa.Company
	UserMigrated bool `json:"migrated"`

	HasTickets bool `json:"has_tickets"`
}

func (m *RootModel) getUsersToMigrate(org *orgMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		users, err := m.client.ZendeskClient.GetOrganizationUsers(ctx, org.ZendeskOrg.Id)
		if err != nil {
			slog.Error("getting users for org", "orgName", org.ZendeskOrg.Name, "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("couldn't get users for org %s: %v", org.ZendeskOrg.Name, err)))
			m.orgsCheckedForUsers++
			m.totalErrors++
			return nil
		}

		for _, user := range users {
			slog.Debug("got user", "orgName", org.ZendeskOrg.Name, "userName", user.Name)
			idString := fmt.Sprintf("%d", user.Id)
			if _, ok := m.data.UsersInPsa[idString]; !ok {
				slog.Debug("adding user to org", "orgName", org.ZendeskOrg.Name, "userName", user.Name)
				m.addUserToUsersMap(idString, &userMigrationDetails{ZendeskUser: &user, PsaCompany: org.PsaOrg})
			} else {
				slog.Debug("user already in org", "orgName", org.ZendeskOrg.Name, "userName", user.Name)
			}
		}

		slog.Info("got users for org", "orgName", org.ZendeskOrg.Name, "totalUsers", len(org.UsersToMigrate))
		m.orgsCheckedForUsers++
		return nil
	}
}

func (m *RootModel) addUserToUsersMap(idString string, user *userMigrationDetails) {
	m.data.UsersToMigrate[idString] = user
}

func (m *RootModel) migrateUsers(user *userMigrationDetails) tea.Cmd {
	return func() tea.Msg {
		if user.UserMigrated {
			slog.Debug("user already migrated", "userEmail", user.ZendeskUser.Email)
			m.totalUsersProcessed++
			return nil
		}

		if user.ZendeskUser.Email == "" {
			slog.Warn("zendesk user has no email address - skipping", "userName", user.ZendeskUser.Name)
			m.totalUsersProcessed++
			m.totalUsersSkipped++
			return nil
		}

		var err error
		user.PsaContact, err = m.matchZdUserToCwContact(user.ZendeskUser)
		if err != nil {

			if errors.Is(err, psa.NoUserFoundErr{}) {
				slog.Debug("user does not exist in psa - attempting to create new user", "userEmail", user.ZendeskUser.Email)
				user.PsaContact, err = m.createPsaContact(user)

				if err != nil {
					slog.Error("creating user", "userName", user.ZendeskUser.Email, "error", err)
					m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("creating user %s: %v", user.ZendeskUser.Email, err)))
					m.totalUsersProcessed++
					m.totalErrors++
					return nil
				}

				slog.Debug("matched zendesk user to psa user", "userEmail", user.ZendeskUser.Email)
				m.totalNewUsersCreated++

			} else {
				slog.Error("matching zendesk user to psa user", "userEmail", user.ZendeskUser.Email, "error", err)
				m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("matching zendesk user to psa user %s: %v", user.ZendeskUser.Email, err)))
				m.totalUsersProcessed++
				m.totalErrors++
				return nil
			}
		}

		if err := m.updateContactFieldValue(user); err != nil {
			if errors.Is(err, ZendeskFieldAlreadySetErr{}) {
				user.UserMigrated = true
				slog.Debug("zendesk user already has psa contact id field", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
				m.totalUsersProcessed++
				return nil
			}

			slog.Error("updating user contact field value in zendesk", "userEmail", user.ZendeskUser.Email, "error", err)
			m.data.writeToOutput(badRedOutput("ERROR", fmt.Errorf("updating contact field in zendesk for %s: %v", user.ZendeskUser.Email, err)))
			m.totalUsersProcessed++
			m.totalErrors++
			return nil
		}

		if user.PsaContact != nil && user.ZendeskUser.UserFields.PSAContactId == user.PsaContact.Id {
			user.UserMigrated = true
			slog.Info("user is fully migrated", "userEmail", user.ZendeskUser.Email, "psaContactId", user.PsaContact.Id)
			m.data.writeToOutput(goodGreenOutput("SUCCESS", fmt.Sprintf("User fully migrated: %s", user.ZendeskUser.Email)))
			m.totalUsersProcessed++
		}
		return nil
	}

}

func (m *RootModel) matchZdUserToCwContact(user *zendesk.User) (*psa.Contact, error) {
	contact, err := m.client.CwClient.GetContactByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}
	return contact, nil
}

func (m *RootModel) createPsaContact(user *userMigrationDetails) (*psa.Contact, error) {
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
			CommunicationType: "Email",
		},
	}

	return m.client.CwClient.PostContact(ctx, c)
}

type ZendeskFieldAlreadySetErr struct{}

func (e ZendeskFieldAlreadySetErr) Error() string {
	return "zendesk user already has psa contact id field"
}

func (m *RootModel) updateContactFieldValue(user *userMigrationDetails) error {
	if user.ZendeskUser.UserFields.PSAContactId == user.PsaContact.Id {
		slog.Debug("zendesk user already has PSA contact id field",
			"userEmail", user.ZendeskUser.Email,
			"psaContactId", user.PsaContact.Id,
		)
		return ZendeskFieldAlreadySetErr{}
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
